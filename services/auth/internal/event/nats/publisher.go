package nats

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
)

// Publisher implements port.EventPublisher via NATS Core.
// In Phase 1 this was a noop; here we do real best-effort publish.
// Assessment-svc uses the Outbox+relay pattern on top of this for critical events.
type Publisher struct {
	nc *nats.Conn
}

// New connects to NATS and returns a Publisher.
func New(url string) (*Publisher, error) {
	nc, err := nats.Connect(url,
		nats.Name("auth-svc-v2"),
		nats.MaxReconnects(10),
		nats.ReconnectWait(nats.DefaultReconnectWait),
	)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	return &Publisher{nc: nc}, nil
}

// Publish sends payload to the given subject (best-effort, no ACK).
func (p *Publisher) Publish(_ context.Context, subject string, payload []byte) error {
	return p.nc.Publish(subject, payload)
}

// Close drains pending messages and closes the connection.
func (p *Publisher) Close() {
	_ = p.nc.Drain()
}
