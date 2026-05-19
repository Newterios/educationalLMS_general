package port

import (
	"context"

	"github.com/Newterios/lms-system-prob/services/course/internal/model"
	"github.com/google/uuid"
)

type EnrollmentFilter struct {
	CourseID  *uuid.UUID
	StudentID *uuid.UUID
}

type EnrollmentRepository interface {
	Create(ctx context.Context, enrollment *model.Enrollment) error
	Delete(ctx context.Context, courseID, studentID uuid.UUID) error
	List(ctx context.Context, filter EnrollmentFilter, p model.Pagination) ([]*model.Enrollment, int64, error)
}
