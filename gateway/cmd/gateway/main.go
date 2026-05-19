// api-gateway — HTTP/JSON ↔ gRPC bridge for all three microservices.
// Listens on :9080 (configurable via GATEWAY_HTTP_PORT).
// Routes:
//   /api/v1/auth/*        → auth-svc-v2      :50051
//   /api/v1/courses/*     → course-svc-v2    :50052
//   /api/v1/assessments/* → assessment-svc-v2 :50053
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	authv1 "github.com/Newterios/lms-system-prob/proto/auth/v1"
	coursev1 "github.com/Newterios/lms-system-prob/proto/course/v1"
	assessmentv1 "github.com/Newterios/lms-system-prob/proto/assessment/v1"
	"github.com/Newterios/lms-system-prob/gateway/internal/routes"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	port := envOr("GATEWAY_HTTP_PORT", "9080")
	authTarget := envOr("GATEWAY_AUTH_TARGET", "localhost:50051")
	courseTarget := envOr("GATEWAY_COURSE_TARGET", "localhost:50052")
	assessmentTarget := envOr("GATEWAY_ASSESSMENT_TARGET", "localhost:50053")

	// ── gRPC clients ──────────────────────────────────────────────────────────
	authConn := mustDial(authTarget)
	defer authConn.Close()

	courseConn := mustDial(courseTarget)
	defer courseConn.Close()

	assessmentConn := mustDial(assessmentTarget)
	defer assessmentConn.Close()

	// ── HTTP router ───────────────────────────────────────────────────────────
	mux := http.NewServeMux()

	routes.RegisterAuth(mux, authv1.NewAuthServiceClient(authConn))
	routes.RegisterCourse(mux, coursev1.NewCourseServiceClient(courseConn))
	routes.RegisterAssessment(mux, assessmentv1.NewAssessmentServiceClient(assessmentConn))

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, `{"status":"ok"}`)
	})

	// ── serve ─────────────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      corsMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	go func() {
		slog.Info("api-gateway listening", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down api-gateway")
	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutCtx)
	slog.Info("done")
}

func mustDial(target string) *grpc.ClientConn {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("dial gRPC", "target", target, "err", err)
		os.Exit(1)
	}
	return conn
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// corsMiddleware adds permissive CORS headers so the Next.js frontend can call the gateway.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,PUT,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
