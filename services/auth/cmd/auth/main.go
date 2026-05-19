package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

	authv1 "github.com/Newterios/lms-system-prob/proto/auth/v1"
	"github.com/Newterios/lms-system-prob/services/auth/internal/app"
	"github.com/Newterios/lms-system-prob/services/auth/internal/cache/noop"
	rediscache "github.com/Newterios/lms-system-prob/services/auth/internal/cache/redis"
	systemclock "github.com/Newterios/lms-system-prob/services/auth/internal/clock/system"
	uuidcode "github.com/Newterios/lms-system-prob/services/auth/internal/code/uuid"
	noopevent "github.com/Newterios/lms-system-prob/services/auth/internal/event/noop"
	natsevent "github.com/Newterios/lms-system-prob/services/auth/internal/event/nats"
	bcrypthasher "github.com/Newterios/lms-system-prob/services/auth/internal/hasher/bcrypt"
	logmailer "github.com/Newterios/lms-system-prob/services/auth/internal/mailer/log"
	smtpmailer "github.com/Newterios/lms-system-prob/services/auth/internal/mailer/smtp"
	"github.com/Newterios/lms-system-prob/services/auth/internal/repository/postgres"
	jwttoken "github.com/Newterios/lms-system-prob/services/auth/internal/token/jwt"
	transportgrpc "github.com/Newterios/lms-system-prob/services/auth/internal/transport/grpc"
	"github.com/Newterios/lms-system-prob/services/auth/internal/transport/grpc/interceptors"
	"github.com/Newterios/lms-system-prob/services/auth/internal/usecase"
	"github.com/Newterios/lms-system-prob/services/auth/internal/otelsetup"
	"github.com/Newterios/lms-system-prob/services/auth/internal/usecase/port"

	"github.com/jackc/pgx/v5/pgxpool"
)



type config struct {
	GRPCPort         string
	DBURL            string
	MigrationsDir    string
	JWTAccessSecret  string
	JWTRefreshSecret string
	JWTAccessTTL     time.Duration
	JWTRefreshTTL    time.Duration
}

func loadConfig() config {
	return config{
		GRPCPort:         envOr("GRPC_PORT", "50051"),
		DBURL:            mustEnv("DB_URL"),
		MigrationsDir:    envOr("MIGRATIONS_DIR", "services/auth/migrations"),
		JWTAccessSecret:  mustEnv("JWT_ACCESS_SECRET"),
		JWTRefreshSecret: mustEnv("JWT_REFRESH_SECRET"),
		JWTAccessTTL:     parseDuration(envOr("JWT_ACCESS_TTL", "15m")),
		JWTRefreshTTL:    parseDuration(envOr("JWT_REFRESH_TTL", "168h")),
	}
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := loadConfig()

	// ── OTel ────────────────────────────────────────────────────────────────
	otelShutdown, err := otelsetup.Setup(context.Background(), "auth-svc-v2")
	if err != nil {
		slog.Warn("OTel init failed (tracing disabled)", "err", err)
	}
	defer otelShutdown()

	// ── migrations ──────────────────────────────────────────────────────────
	if err := app.RunMigrations(cfg.DBURL, cfg.MigrationsDir); err != nil {
		slog.Error("migrations failed", "err", err)
		os.Exit(1)
	}

	// ── database pool ────────────────────────────────────────────────────────
	pool, err := pgxpool.New(context.Background(), cfg.DBURL)
	if err != nil {
		slog.Error("connect to postgres", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	// ── infrastructure ────────────────────────────────────────────────────────
	hasher := bcrypthasher.New(bcrypthasher.DefaultCost)
	signer := jwttoken.New(jwttoken.Config{
		AccessSecret:  []byte(cfg.JWTAccessSecret),
		RefreshSecret: []byte(cfg.JWTRefreshSecret),
		AccessTTL:     cfg.JWTAccessTTL,
		RefreshTTL:    cfg.JWTRefreshTTL,
	})
	codeGen := uuidcode.New()
	clock := systemclock.New()
	// ── cache (Redis if REDIS_URL set, otherwise noop) ─────────────────────────
	var cacheImpl interface {
		port.Cache
	}
	if redisURL := envOr("REDIS_URL", ""); redisURL != "" {
		rc, err := rediscache.New(redisURL)
		if err != nil {
			slog.Warn("Redis connect failed, falling back to noop cache", "err", err)
			cacheImpl = noop.New()
		} else {
			cacheImpl = rc
			slog.Info("Redis cache connected", "url", redisURL)
		}
	} else {
		cacheImpl = noop.New()
		slog.Warn("REDIS_URL not set — using noop cache")
	}
	cache := cacheImpl

	// ── NATS publisher (falls back to noop if NATS_URL not set) ───────────────
	var events port.EventPublisher
	var natsCloser func()
	if natsURL := envOr("NATS_URL", ""); natsURL != "" {
		natsPub, err := natsevent.New(natsURL)
		if err != nil {
			slog.Warn("NATS connect failed, falling back to noop publisher", "err", err)
			events = noopevent.New()
		} else {
			events = natsPub
			natsCloser = natsPub.Close
			slog.Info("NATS publisher connected", "url", natsURL)
		}
	} else {
		events = noopevent.New()
		slog.Warn("NATS_URL not set — using noop event publisher")
	}
	// ── mailer ────────────────────────────────────────────────────────────────
	var mailer port.Mailer
	if envOr("MAILER", "mock") == "smtp" {
		sm := smtpmailer.New(smtpmailer.ConfigFromEnv())
		if err := sm.Ping(); err != nil {
			slog.Warn("SMTP server unreachable, falling back to logmailer", "err", err)
			mailer = logmailer.New()
		} else {
			mailer = sm
			slog.Info("SMTP mailer connected", "host", smtpmailer.ConfigFromEnv().Host)
		}
	} else {
		mailer = logmailer.New()
		slog.Info("logmailer active (set MAILER=smtp to use real SMTP)")
	}

	// ── repositories ─────────────────────────────────────────────────────────
	userRepo := postgres.NewUserRepository(pool)
	sessionRepo := postgres.NewSessionRepository(pool)
	codeRepo := postgres.NewVerificationCodeRepository(pool)
	txRunner := postgres.NewTxRunner(pool)

	// ── use-cases ─────────────────────────────────────────────────────────────
	registerUC := usecase.NewRegisterUseCase(userRepo, codeRepo, codeGen, hasher, events, mailer, clock)
	loginUC := usecase.NewLoginUseCase(userRepo, sessionRepo, hasher, signer, cache, clock)
	refreshTokenUC := usecase.NewRefreshTokenUseCase(userRepo, sessionRepo, signer, cache, clock)
	logoutUC := usecase.NewLogoutUseCase(sessionRepo, events, clock)
	verifyEmailUC := usecase.NewVerifyEmailUseCase(codeRepo, userRepo, events, clock)
	requestPwdResetUC := usecase.NewRequestPasswordResetUseCase(userRepo, codeRepo, events, mailer, codeGen, clock)
	confirmPwdResetUC := usecase.NewConfirmPasswordResetUseCase(codeRepo, userRepo, sessionRepo, hasher, events, txRunner, clock)
	changePwdUC := usecase.NewChangePasswordUseCase(userRepo, sessionRepo, hasher, events, txRunner, clock)
	getMeUC := usecase.NewGetMeUseCase(userRepo, cache)
	updateProfileUC := usecase.NewUpdateProfileUseCase(userRepo, cache, events, clock)
	listSessionsUC := usecase.NewListSessionsUseCase(sessionRepo)
	revokeSessionUC := usecase.NewRevokeSessionUseCase(sessionRepo, events, clock)

	// ── gRPC server ───────────────────────────────────────────────────────────
	publicMethods := map[string]bool{
		authv1.AuthService_Register_FullMethodName:             true,
		authv1.AuthService_Login_FullMethodName:                true,
		authv1.AuthService_RefreshToken_FullMethodName:         true,
		authv1.AuthService_Logout_FullMethodName:               true, // refresh_token proves identity; access token may be expired
		authv1.AuthService_VerifyEmail_FullMethodName:          true,
		authv1.AuthService_RequestPasswordReset_FullMethodName: true,
		authv1.AuthService_ConfirmPasswordReset_FullMethodName: true,
	}
	for k, v := range interceptors.InfraPublicMethods {
		publicMethods[k] = v
	}

	// build interceptor chain: Recovery → Auth → RateLimit (if Redis) → Logging
	unaryChain := []grpc.UnaryServerInterceptor{
		interceptors.Recovery(),
		interceptors.Auth(signer, publicMethods),
	}
	if rl, ok := cache.(interceptors.RateLimiter); ok {
		unaryChain = append(unaryChain, interceptors.RateLimit(rl, interceptors.RateLimitConfig{
			GlobalRPM: 100,
			MethodLimits: map[string]int64{
				authv1.AuthService_Login_FullMethodName: 10,
			},
		}))
	}
	unaryChain = append(unaryChain, interceptors.Logging())

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(unaryChain...),
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)


	svc := transportgrpc.NewServer(
		registerUC, loginUC, refreshTokenUC, logoutUC,
		verifyEmailUC, requestPwdResetUC, confirmPwdResetUC, changePwdUC,
		getMeUC, updateProfileUC, listSessionsUC, revokeSessionUC,
	)
	authv1.RegisterAuthServiceServer(grpcServer, svc)

	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	grpc_health_v1.RegisterHealthServer(grpcServer, healthSrv)

	reflection.Register(grpcServer)

	// ── listen ────────────────────────────────────────────────────────────────
	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		slog.Error("listen", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	go func() {
		slog.Info("auth-svc-v2 listening", "port", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("serve", "err", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down gracefully")

	// close NATS before gRPC drain
	if natsCloser != nil {
		natsCloser()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		slog.Info("graceful shutdown complete")
	case <-shutdownCtx.Done():
		slog.Warn("graceful shutdown timed out, forcing stop")
		grpcServer.Stop()
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fmt.Fprintf(os.Stderr, "fatal: required env var %s is not set\n", key)
		os.Exit(1)
	}
	return v
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: invalid duration %q: %v\n", s, err)
		os.Exit(1)
	}
	return d
}
