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

	"github.com/jackc/pgx/v5/pgxpool"

	assessmentv1 "github.com/Newterios/lms-system-prob/proto/assessment/v1"
	"github.com/Newterios/lms-system-prob/services/assessment/internal/app"
	noopCache "github.com/Newterios/lms-system-prob/services/assessment/internal/cache/noop"
	redisCache "github.com/Newterios/lms-system-prob/services/assessment/internal/cache/redis"
	courseclient "github.com/Newterios/lms-system-prob/services/assessment/internal/client/course"
	noopevent "github.com/Newterios/lms-system-prob/services/assessment/internal/event/noop"
	natsevent "github.com/Newterios/lms-system-prob/services/assessment/internal/event/nats"
	"github.com/Newterios/lms-system-prob/services/assessment/internal/otelsetup"
	"github.com/Newterios/lms-system-prob/services/assessment/internal/outbox"
	"github.com/Newterios/lms-system-prob/services/assessment/internal/repository/postgres"
	jwttoken "github.com/Newterios/lms-system-prob/services/assessment/internal/token/jwt"
	transportgrpc "github.com/Newterios/lms-system-prob/services/assessment/internal/transport/grpc"
	"github.com/Newterios/lms-system-prob/services/assessment/internal/transport/grpc/interceptors"
	"github.com/Newterios/lms-system-prob/services/assessment/internal/usecase"
	"github.com/Newterios/lms-system-prob/services/assessment/internal/usecase/port"
)

type config struct {
	GRPCPort          string
	DBURL             string
	MigrationsDir     string
	JWTAccessSecret   string
	CourseGRPCTarget  string
}

func loadConfig() config {
	return config{
		GRPCPort:         envOr("ASSESSMENT_GRPC_PORT", "50053"),
		DBURL:            mustEnv("DATABASE_URL_ASSESSMENT"),
		MigrationsDir:    envOr("ASSESSMENT_MIGRATIONS_DIR", "services/assessment/migrations"),
		JWTAccessSecret:  mustEnv("JWT_ACCESS_SECRET"),
		CourseGRPCTarget: envOr("COURSE_GRPC_TARGET", "localhost:50052"),
	}
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := loadConfig()

	// ── OTel ────────────────────────────────────────────────────────────────
	otelShutdown, err := otelsetup.Setup(context.Background(), "assessment-svc-v2")
	if err != nil {
		slog.Warn("OTel init failed (tracing disabled)", "err", err)
	}
	defer otelShutdown()

	// ── migrations ────────────────────────────────────────────────────────────
	if err := app.RunMigrations(cfg.DBURL, cfg.MigrationsDir); err != nil {
		slog.Error("migrations failed", "err", err)
		os.Exit(1)
	}

	// ── database ──────────────────────────────────────────────────────────────
	pool, err := pgxpool.New(context.Background(), cfg.DBURL)
	if err != nil {
		slog.Error("connect to postgres", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	// ── infrastructure ────────────────────────────────────────────────────────
	signer := jwttoken.New(jwttoken.Config{AccessSecret: []byte(cfg.JWTAccessSecret)})

	// ── cache ────────────────────────────────────────────────────────────────
	var cache port.Cache
	if redisURL := envOr("REDIS_URL", ""); redisURL != "" {
		rc, err := redisCache.New(redisURL)
		if err != nil {
			slog.Warn("Redis connect failed, falling back to noop", "err", err)
			cache = noopCache.New()
		} else {
			cache = rc
			slog.Info("Redis cache connected", "url", redisURL)
		}
	} else {
		cache = noopCache.New()
		slog.Warn("REDIS_URL not set — using noop cache")
	}

	// ── NATS publisher (relay goroutine + best-effort events) ─────────────────
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
	_ = natsCloser // used in shutdown

	// ── cross-service client ──────────────────────────────────────────────────
	courseClient, err := courseclient.New(cfg.CourseGRPCTarget, slog.Default())
	if err != nil {
		slog.Error("dial course-svc", "err", err)
		os.Exit(1)
	}

	// ── repositories ─────────────────────────────────────────────────────────
	quizRepo := postgres.NewQuizRepository(pool)
	attemptRepo := postgres.NewAttemptRepository(pool)
	outboxRepo := postgres.NewOutboxRepository(pool)
	txRunner := postgres.NewTxRunner(pool)

	// ── use-cases ─────────────────────────────────────────────────────────────
	createQuizUC := usecase.NewCreateQuizUseCase(quizRepo, outboxRepo, txRunner)
	getQuizUC := usecase.NewGetQuizUseCase(quizRepo, cache)
	updateQuizUC := usecase.NewUpdateQuizUseCase(quizRepo, outboxRepo, cache, txRunner)
	deleteQuizUC := usecase.NewDeleteQuizUseCase(quizRepo, attemptRepo, outboxRepo, cache, txRunner)
	listQuizzesUC := usecase.NewListQuizzesUseCase(quizRepo, cache)
	startAttemptUC := usecase.NewStartAttemptUseCase(quizRepo, attemptRepo, courseClient, events)
	submitAttemptUC := usecase.NewSubmitAttemptUseCase(quizRepo, attemptRepo, outboxRepo, txRunner)
	getAttemptUC := usecase.NewGetAttemptUseCase(attemptRepo, quizRepo)
	listAttemptsUC := usecase.NewListAttemptsUseCase(attemptRepo)
	gradeSubmissionUC := usecase.NewGradeSubmissionUseCase(attemptRepo, quizRepo, outboxRepo, cache, txRunner)
	getGradebookUC := usecase.NewGetGradebookUseCase(attemptRepo, quizRepo, cache)
	exportGradesUC := usecase.NewExportGradesUseCase(attemptRepo, quizRepo, cache)

	// ── gRPC server ───────────────────────────────────────────────────────────
	publicMethods := make(map[string]bool)
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
				assessmentv1.AssessmentService_SubmitAttempt_FullMethodName: 5,
				assessmentv1.AssessmentService_StartAttempt_FullMethodName:  20,
			},
		}))
	}
	unaryChain = append(unaryChain, interceptors.Logging())

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(unaryChain...),
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)

	svc := transportgrpc.NewServer(
		createQuizUC, getQuizUC, updateQuizUC, deleteQuizUC, listQuizzesUC,
		startAttemptUC, submitAttemptUC, getAttemptUC, listAttemptsUC,
		gradeSubmissionUC, getGradebookUC, exportGradesUC,
	)
	assessmentv1.RegisterAssessmentServiceServer(grpcServer, svc)

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

	// ── outbox relay ──────────────────────────────────────────────────────────
	relay := outbox.NewRelay(outboxRepo, events, outbox.Config{}, slog.Default())
	go func() {
		if err := relay.Run(ctx); err != nil && err != context.Canceled {
			slog.Error("relay exited", "err", err)
		}
	}()

	// ── serve ─────────────────────────────────────────────────────────────────
	go func() {
		slog.Info("assessment-svc-v2 listening", "port", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("serve", "err", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down gracefully")

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
