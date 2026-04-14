package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Newterios/lms-system-prob/services/assessment/internal/model"
	"github.com/google/uuid"
)

type outboxRepository struct{ pool *pgxpool.Pool }

func NewOutboxRepository(pool *pgxpool.Pool) *outboxRepository {
	return &outboxRepository{pool: pool}
}

func (r *outboxRepository) Insert(ctx context.Context, e *model.OutboxEntry) error {
	_, err := db(ctx, r.pool).Exec(ctx, `
		INSERT INTO outbox (aggregate_id, event_type, payload, occurred_at)
		VALUES ($1,$2,$3,$4)`,
		e.AggregateID, e.EventType, e.Payload, e.OccurredAt,
	)
	if err != nil {
		return fmt.Errorf("outbox insert: %w", err)
	}
	return nil
}

func (r *outboxRepository) ListUnpublished(ctx context.Context, limit int) ([]*model.OutboxEntry, error) {
	rows, err := db(ctx, r.pool).Query(ctx, `
		SELECT id, aggregate_id, event_type, payload, occurred_at, published_at
		FROM outbox
		WHERE published_at IS NULL
		ORDER BY id
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("outbox list unpublished: %w", err)
	}
	defer rows.Close()

	var entries []*model.OutboxEntry
	for rows.Next() {
		var e model.OutboxEntry
		if err := rows.Scan(&e.ID, &e.AggregateID, &e.EventType, &e.Payload, &e.OccurredAt, &e.PublishedAt); err != nil {
			return nil, fmt.Errorf("scan outbox: %w", err)
		}
		entries = append(entries, &e)
	}
	return entries, rows.Err()
}

func (r *outboxRepository) MarkPublished(ctx context.Context, id int64, publishedAt time.Time) error {
	_, err := db(ctx, r.pool).Exec(ctx,
		`UPDATE outbox SET published_at=$1 WHERE id=$2`, publishedAt, id)
	if err != nil {
		return fmt.Errorf("outbox mark published: %w", err)
	}
	return nil
}

// InsertWithAggregateID is a helper to build + insert an OutboxEntry.
func InsertOutboxEntry(ctx context.Context, repo interface {
	Insert(context.Context, *model.OutboxEntry) error
}, aggregateID uuid.UUID, eventType string, payload []byte) error {
	return repo.Insert(ctx, &model.OutboxEntry{
		AggregateID: aggregateID,
		EventType:   eventType,
		Payload:     payload,
		OccurredAt:  time.Now(),
	})
}
