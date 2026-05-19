// notification-svc-v2 — NATS subscriber + AP4 worker pool + dead-letter.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	goredis "github.com/redis/go-redis/v9"

	"github.com/Newterios/lms-system-prob/notification/internal/jobqueue"
	"github.com/Newterios/lms-system-prob/notification/internal/subscriber"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	natsURL := mustEnv("NATS_URL")
	gatewayURL := envOr("GATEWAY_URL", "http://localhost:8090")
	poolSize := parseInt(envOr("WORKER_POOL_SIZE", "3"), 3)
	maxRetries := parseInt(envOr("JOB_RETRY_ATTEMPTS", "3"), 3)
	dlqKey := envOr("DLQ_KEY", "dlq:notification")

	slog.Info("notification-svc-v2 starting",
		"nats", natsURL,
		"gateway", gatewayURL,
		"pool_size", poolSize,
		"max_retries", maxRetries,
	)

	// ── Redis DLQ writer ────────────────────────────────────────────────────────────────
	var dlq jobqueue.DLQWriter
	if redisURL := envOr("REDIS_URL", ""); redisURL != "" {
		opts, err := goredis.ParseURL(redisURL)
		if err != nil {
			slog.Warn("Redis URL parse failed, DLQ will go to stderr only", "err", err)
		} else {
			rc := goredis.NewClient(opts)
			dlq = &redisDLQ{rc}
			slog.Info("Redis DLQ connected", "key", dlqKey)
		}
	} else {
		slog.Warn("REDIS_URL not set — DLQ entries go to stderr only")
	}

	pool := jobqueue.New(jobqueue.Config{
		GatewayURL: gatewayURL,
		PoolSize:   poolSize,
		MaxRetries: maxRetries,
		DLQKey:     dlqKey,
	}, dlq, slog.Default())

	sub, err := subscriber.New(natsURL, pool, slog.Default())
	if err != nil {
		slog.Error("NATS subscriber failed", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	slog.Info("notification-svc-v2 ready")
	<-ctx.Done()
	slog.Info("shutting down")
	sub.Drain(context.Background())
	slog.Info("done")
}

// redisDLQ wraps go-redis to implement jobqueue.DLQWriter.
type redisDLQ struct{ client *goredis.Client }

func (r *redisDLQ) LPush(ctx context.Context, key, value string) error {
	return r.client.LPush(ctx, key, value).Err()
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

func parseInt(s string, fallback int) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return v
}
