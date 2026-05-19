package noop

import (
	"context"
	"log/slog"
)

type Publisher struct{}

func New() *Publisher { return &Publisher{} }

func (p *Publisher) Publish(_ context.Context, subject string, payload []byte) error {
	slog.Debug("noop event publisher", "subject", subject, "payload_bytes", len(payload))
	return nil
}
