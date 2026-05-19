package port

import "context"

// EventPublisher publishes domain events to NATS (best-effort, fire-and-forget).
// If the broker is unavailable the RPC still succeeds; the error is logged only.
type EventPublisher interface {
	Publish(ctx context.Context, subject string, payload []byte) error
}
