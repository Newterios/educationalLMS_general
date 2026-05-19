package port

import (
	"context"

	"github.com/Newterios/lms-system-prob/services/course/internal/model"
	"github.com/google/uuid"
)

type MaterialRepository interface {
	Create(ctx context.Context, material *model.Material) error
	ListBySectionID(ctx context.Context, sectionID uuid.UUID) ([]*model.Material, error)
}
