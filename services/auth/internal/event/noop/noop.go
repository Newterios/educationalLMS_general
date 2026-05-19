package noop

import (
	"context"
	"log/slog"
)

// Publisher is a no-op EventPublisher that logs the would-be event.
// Used in Phase 1C. Real NATS publisher replaces it in Phase 2.
type Publisher struct{}

func New() *Publisher { return &Publisher{} }

func (p *Publisher) Publish(_ context.Context, subject string, payload []byte) error {
	slog.Debug("noop event publisher", "subject", subject, "payload_bytes", len(payload))
	return nil
}
