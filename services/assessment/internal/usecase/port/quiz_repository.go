package port

import (
	"context"

	"github.com/Newterios/lms-system-prob/services/assessment/internal/model"
	"github.com/google/uuid"
)

type QuizRepository interface {
	Create(ctx context.Context, quiz *model.Quiz) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Quiz, error)
	Update(ctx context.Context, quiz *model.Quiz) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByCourseID(ctx context.Context, courseID uuid.UUID, p model.Pagination) ([]*model.Quiz, int64, error)
}
