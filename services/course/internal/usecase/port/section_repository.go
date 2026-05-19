package port

import (
	"context"

	"github.com/Newterios/lms-system-prob/services/course/internal/model"
	"github.com/google/uuid"
)

type SectionRepository interface {
	Create(ctx context.Context, section *model.Section) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Section, error)
	ListByCourseID(ctx context.Context, courseID uuid.UUID) ([]*model.Section, error)
}
