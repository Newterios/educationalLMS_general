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

	coursev1 "github.com/Newterios/lms-system-prob/proto/course/v1"
	"github.com/Newterios/lms-system-prob/services/course/internal/app"
	"github.com/Newterios/lms-system-prob/services/course/internal/cache/noop"
	rediscache "github.com/Newterios/lms-system-prob/services/course/internal/cache/redis"
	systemclock "github.com/Newterios/lms-system-prob/services/course/internal/clock/system"
	noopevent "github.com/Newterios/lms-system-prob/services/course/internal/event/noop"
	natsevent "github.com/Newterios/lms-system-prob/services/course/internal/event/nats"
	"github.com/Newterios/lms-system-prob/services/course/internal/otelsetup"
	"github.com/Newterios/lms-system-prob/services/course/internal/repository/postgres"
	jwttoken "github.com/Newterios/lms-system-prob/services/course/internal/token/jwt"
	transportgrpc "github.com/Newterios/lms-system-prob/services/course/internal/transport/grpc"
	"github.com/Newterios/lms-system-prob/services/course/internal/transport/grpc/interceptors"
	"github.com/Newterios/lms-system-prob/services/course/internal/usecase"
	"github.com/Newterios/lms-system-prob/services/course/internal/usecase/port"
)

type config struct {
	GRPCPort        string
	DBURL           string
	MigrationsDir   string
	JWTAccessSecret string
}

func loadConfig() config {
	return config{
		GRPCPort:        envOr("COURSE_GRPC_PORT", "50052"),
		DBURL:           mustEnv("DATABASE_URL_COURSE"),
		MigrationsDir:   envOr("COURSE_MIGRATIONS_DIR", "services/course/migrations"),
		JWTAccessSecret: mustEnv("JWT_ACCESS_SECRET"),
	}
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := loadConfig()

	// ── OTel ────────────────────────────────────────────────────────────────
	otelShutdown, err := otelsetup.Setup(context.Background(), "course-svc-v2")
	if err != nil {
		slog.Warn("OTel init failed (tracing disabled)", "err", err)
	}
	defer otelShutdown()

	if err := app.RunMigrations(cfg.DBURL, cfg.MigrationsDir); err != nil {
		slog.Error("migrations failed", "err", err)
		os.Exit(1)
	}

	pool, err := pgxpool.New(context.Background(), cfg.DBURL)
	if err != nil {
		slog.Error("connect to postgres", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	// ── infrastructure ────────────────────────────────────────────────────────
	signer := jwttoken.New(jwttoken.Config{AccessSecret: []byte(cfg.JWTAccessSecret)})
	clock := systemclock.New()
	// ── cache ────────────────────────────────────────────────────────────────
	var cache port.Cache
	if redisURL := envOr("REDIS_URL", ""); redisURL != "" {
		rc, err := rediscache.New(redisURL)
		if err != nil {
			slog.Warn("Redis connect failed, falling back to noop", "err", err)
			cache = noop.New()
		} else {
			cache = rc
			slog.Info("Redis cache connected", "url", redisURL)
		}
	} else {
		cache = noop.New()
		slog.Warn("REDIS_URL not set — using noop cache")
	}

	// ── NATS publisher ────────────────────────────────────────────────────────
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

	// ── repositories ──────────────────────────────────────────────────────────
	courseRepo := postgres.NewCourseRepository(pool)
	sectionRepo := postgres.NewSectionRepository(pool)
	materialRepo := postgres.NewMaterialRepository(pool)
	enrollmentRepo := postgres.NewEnrollmentRepository(pool)

	// ── use-cases ─────────────────────────────────────────────────────────────
	createCourseUC := usecase.NewCreateCourseUseCase(courseRepo, cache, events, clock)
	getCourseUC := usecase.NewGetCourseUseCase(courseRepo, cache)
	updateCourseUC := usecase.NewUpdateCourseUseCase(courseRepo, cache, events, clock)
	deleteCourseUC := usecase.NewDeleteCourseUseCase(courseRepo, cache, events, clock)
	listCoursesUC := usecase.NewListCoursesUseCase(courseRepo, cache)
	createSectionUC := usecase.NewCreateSectionUseCase(sectionRepo, courseRepo, cache, events)
	listSectionsUC := usecase.NewListSectionsUseCase(sectionRepo, cache)
	addMaterialUC := usecase.NewAddMaterialUseCase(materialRepo, sectionRepo, courseRepo, cache, events)
	listMaterialsUC := usecase.NewListMaterialsUseCase(materialRepo, cache)
	enrollStudentUC := usecase.NewEnrollStudentUseCase(enrollmentRepo, courseRepo, events, clock)
	unenrollStudentUC := usecase.NewUnenrollStudentUseCase(enrollmentRepo, courseRepo, events)
	listEnrollmentsUC := usecase.NewListEnrollmentsUseCase(enrollmentRepo)

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
				coursev1.CourseService_EnrollStudent_FullMethodName: 20,
			},
		}))
	}
	unaryChain = append(unaryChain, interceptors.Logging())

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(unaryChain...),
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)

	svc := transportgrpc.NewServer(
		createCourseUC, getCourseUC, updateCourseUC, deleteCourseUC, listCoursesUC,
		createSectionUC, listSectionsUC,
		addMaterialUC, listMaterialsUC,
		enrollStudentUC, unenrollStudentUC, listEnrollmentsUC,
	)
	coursev1.RegisterCourseServiceServer(grpcServer, svc)

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
		slog.Info("course-svc-v2 listening", "port", cfg.GRPCPort)
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
