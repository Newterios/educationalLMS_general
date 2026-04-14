package port

import (
	"context"

	"github.com/Newterios/lms-system-prob/services/assessment/internal/model"
	"github.com/google/uuid"
)

type AttemptFilter struct {
	QuizID    *uuid.UUID
	StudentID *uuid.UUID
}

type AttemptRepository interface {
	Create(ctx context.Context, attempt *model.Attempt) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Attempt, error)
	Update(ctx context.Context, attempt *model.Attempt) error
	List(ctx context.Context, filter AttemptFilter, p model.Pagination) ([]*model.Attempt, int64, error)
	ListByCourseID(ctx context.Context, courseID uuid.UUID) ([]*model.Attempt, error)
	ExistsForQuiz(ctx context.Context, quizID uuid.UUID) (bool, error)
}

