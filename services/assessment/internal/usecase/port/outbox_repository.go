package port

import (
	"context"
	"time"

	"github.com/Newterios/lms-system-prob/services/assessment/internal/model"
)

type OutboxRepository interface {
	Insert(ctx context.Context, entry *model.OutboxEntry) error
	ListUnpublished(ctx context.Context, limit int) ([]*model.OutboxEntry, error)
	MarkPublished(ctx context.Context, id int64, publishedAt time.Time) error
}
