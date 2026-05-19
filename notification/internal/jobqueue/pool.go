package jobqueue

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"time"
)

// Job represents a notification task from a NATS event.
type Job struct {
	EventType string
	EntityID  string    // extracted from payload for idempotency
	OccurredAt time.Time
	Payload   []byte
}

// IdempotencyKey returns SHA256(EventType+EntityID+OccurredAt) — per PLAN.md §Phase 3.
func (j Job) IdempotencyKey() string {
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|%s", j.EventType, j.EntityID, j.OccurredAt.UTC().Format(time.RFC3339Nano))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// GatewayRequest is the JSON body sent to mock-gateway POST /notify.
type GatewayRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	EventType      string `json:"event_type"`
	Payload        string `json:"payload"` // hex-encoded
}

// Config holds worker pool configuration.
type Config struct {
	GatewayURL   string
	PoolSize     int
	MaxRetries   int
	DLQKey       string // Redis LPUSH key for dead-letter
}

// DLQWriter stores dead-letter entries. Implemented by Redis client.
type DLQWriter interface {
	LPush(ctx context.Context, key string, value string) error
}

// Pool is a fixed-size worker pool that dispatches notification jobs.
type Pool struct {
	cfg     Config
	jobs    chan Job
	dlq     DLQWriter
	client  *http.Client
	log     *slog.Logger
}

// New creates a Pool and starts cfg.PoolSize workers.
func New(cfg Config, dlq DLQWriter, log *slog.Logger) *Pool {
	p := &Pool{
		cfg:    cfg,
		jobs:   make(chan Job, cfg.PoolSize*10),
		dlq:    dlq,
		client: &http.Client{Timeout: 5 * time.Second},
		log:    log,
	}
	for i := 0; i < cfg.PoolSize; i++ {
		go p.worker(i)
	}
	return p
}

// Submit enqueues a job (non-blocking, drops if channel full).
func (p *Pool) Submit(j Job) {
	select {
	case p.jobs <- j:
	default:
		p.log.Warn("job queue full, dropping job", "event", j.EventType)
	}
}

func (p *Pool) worker(id int) {
	p.log.Info("notification worker started", "id", id)
	for j := range p.jobs {
		p.dispatch(j)
	}
}

// dispatch calls mock-gateway with exponential backoff, dead-letters on failure.
func (p *Pool) dispatch(j Job) {
	key := j.IdempotencyKey()
	body := GatewayRequest{
		IdempotencyKey: key,
		EventType:      j.EventType,
		Payload:        fmt.Sprintf("%x", j.Payload),
	}
	bodyBytes, _ := json.Marshal(body)

	var lastErr error
	for attempt := 0; attempt <= p.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt-1))*float64(200*time.Millisecond))
			if backoff > 5*time.Second {
				backoff = 5 * time.Second
			}
			time.Sleep(backoff)
		}

		resp, err := p.client.Post(p.cfg.GatewayURL+"/notify", "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			lastErr = err
			p.log.Warn("gateway POST failed", "event", j.EventType, "attempt", attempt, "err", err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			p.log.Info("notification delivered", "event", j.EventType, "key", key, "attempt", attempt)
			return
		}
		lastErr = fmt.Errorf("gateway returned %d", resp.StatusCode)
		p.log.Warn("gateway returned error", "event", j.EventType, "status", resp.StatusCode, "attempt", attempt)
	}

	// dead-letter
	p.log.Error("notification dead-lettered", "event", j.EventType, "key", key, "err", lastErr)
	dlqEntry, _ := json.Marshal(map[string]any{
		"event_type":      j.EventType,
		"idempotency_key": key,
		"payload":         fmt.Sprintf("%x", j.Payload),
		"failed_at":       time.Now().UTC(),
		"error":           lastErr.Error(),
	})
	if p.dlq != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := p.dlq.LPush(ctx, p.cfg.DLQKey, string(dlqEntry)); err != nil {
			p.log.Error("dlq push failed", "err", err)
		}
	}
	// always log to stderr (AP3 §7.2 format)
	fmt.Printf(`{"level":"ERROR","time":"%s","msg":"dead_letter","event_type":"%s","key":"%s"}%s`,
		time.Now().UTC().Format(time.RFC3339), j.EventType, key, "\n")
}
