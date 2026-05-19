package port

import (
	"context"
	"time"

	"github.com/Newterios/lms-system-prob/services/course/internal/model"
	"github.com/google/uuid"
)

type CourseFilter struct {
	TeacherID *uuid.UUID
}

type CourseRepository interface {
	Create(ctx context.Context, course *model.Course) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Course, error)
	Update(ctx context.Context, course *model.Course) error
	SoftDelete(ctx context.Context, id uuid.UUID, deletedAt time.Time) error
	List(ctx context.Context, filter CourseFilter, p model.Pagination) ([]*model.Course, int64, error)
}
