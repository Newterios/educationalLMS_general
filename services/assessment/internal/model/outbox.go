package model

import (
	"time"

	"github.com/google/uuid"
)

type OutboxEntry struct {
	ID          int64
	AggregateID uuid.UUID
	EventType   string
	Payload     []byte // JSON
	OccurredAt  time.Time
	PublishedAt *time.Time
}
