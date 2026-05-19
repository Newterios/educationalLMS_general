package usecase

import (
	"context"

	"github.com/Newterios/lms-system-prob/services/course/internal/model"
	"github.com/Newterios/lms-system-prob/services/course/internal/usecase/port"
	"github.com/google/uuid"
)

type ListEnrollmentsUseCase struct {
	enrollments port.EnrollmentRepository
}

func NewListEnrollmentsUseCase(enrollments port.EnrollmentRepository) *ListEnrollmentsUseCase {
	return &ListEnrollmentsUseCase{enrollments: enrollments}
}

type ListEnrollmentsInput struct {
	CourseID   *uuid.UUID
	StudentID  *uuid.UUID
	Pagination model.Pagination
}

type ListEnrollmentsOutput struct {
	Enrollments []*model.Enrollment
	TotalCount  int64
}

func (uc *ListEnrollmentsUseCase) Execute(ctx context.Context, in ListEnrollmentsInput) (ListEnrollmentsOutput, error) {
	filter := port.EnrollmentFilter{CourseID: in.CourseID, StudentID: in.StudentID}
	enrollments, total, err := uc.enrollments.List(ctx, filter, in.Pagination)
	if err != nil {
		return ListEnrollmentsOutput{}, err
	}
	return ListEnrollmentsOutput{Enrollments: enrollments, TotalCount: total}, nil
}
