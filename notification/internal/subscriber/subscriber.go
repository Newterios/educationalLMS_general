package subscriber

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/Newterios/lms-system-prob/notification/internal/jobqueue"
)

// eventPayload is the canonical shape published by all three gRPC services.
type eventPayload struct {
	EventType  string          `json:"event_type"`
	EntityID   string          `json:"entity_id"`
	OccurredAt time.Time       `json:"occurred_at"`
	Data       json.RawMessage `json:"data"`
}

// Subscriber wraps a NATS connection and routes messages to the job pool.
type Subscriber struct {
	nc   *nats.Conn
	pool *jobqueue.Pool
	log  *slog.Logger
	subs []*nats.Subscription
}

// Subjects to subscribe to — wildcards cover every event from the three services.
var subjects = []string{
	"auth.>",
	"course.>",
	"assessment.>",
}

// New connects to NATS and subscribes to all relevant subjects.
func New(natsURL string, pool *jobqueue.Pool, log *slog.Logger) (*Subscriber, error) {
	nc, err := nats.Connect(natsURL,
		nats.Name("notification-svc-v2"),
		nats.MaxReconnects(20),
		nats.ReconnectWait(nats.DefaultReconnectWait),
	)
	if err != nil {
		return nil, err
	}
	s := &Subscriber{nc: nc, pool: pool, log: log}

	for _, subj := range subjects {
		sub, err := nc.Subscribe(subj, s.handle)
		if err != nil {
			_ = nc.Drain()
			return nil, err
		}
		s.subs = append(s.subs, sub)
		log.Info("subscribed", "subject", subj)
	}
	return s, nil
}

func (s *Subscriber) handle(msg *nats.Msg) {
	var p eventPayload
	if err := json.Unmarshal(msg.Data, &p); err != nil {
		// best-effort: use subject as event type, empty entity
		p.EventType = msg.Subject
		p.OccurredAt = time.Now().UTC()
	}
	if p.EventType == "" {
		p.EventType = msg.Subject
	}
	if p.OccurredAt.IsZero() {
		p.OccurredAt = time.Now().UTC()
	}

	s.log.Info("event received", "subject", msg.Subject, "event", p.EventType)
	s.pool.Submit(jobqueue.Job{
		EventType:  p.EventType,
		EntityID:   p.EntityID,
		OccurredAt: p.OccurredAt,
		Payload:    msg.Data,
	})
}

// Drain unsubscribes and drains the NATS connection gracefully.
func (s *Subscriber) Drain(_ context.Context) {
	_ = s.nc.Drain()
}
