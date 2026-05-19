package event

import (
	"encoding/json"
	"time"
)

// Envelope is the canonical NATS event payload structure.
type Envelope struct {
	EventType  string          `json:"event_type"`
	EntityID   string          `json:"entity_id"`
	OccurredAt time.Time       `json:"occurred_at"`
	Data       json.RawMessage `json:"data"`
}

// Marshal encodes an Envelope to JSON (best-effort, never returns nil).
func Marshal(eventType, entityID string, data any) []byte {
	raw, _ := json.Marshal(data)
	env := Envelope{
		EventType:  eventType,
		EntityID:   entityID,
		OccurredAt: time.Now().UTC(),
		Data:       raw,
	}
	b, _ := json.Marshal(env)
	return b
}
