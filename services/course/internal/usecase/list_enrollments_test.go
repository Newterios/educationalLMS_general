package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/Newterios/lms-system-prob/services/course/internal/model"
	"github.com/Newterios/lms-system-prob/services/course/internal/usecase"
	"github.com/Newterios/lms-system-prob/services/course/internal/usecase/usecasetest"
	"github.com/google/uuid"
)

func TestListEnrollments_ByCourse(t *testing.T) {
	enrollments := usecasetest.NewFakeEnrollmentRepository()
	uc := usecase.NewListEnrollmentsUseCase(enrollments)

	courseID := uuid.New()
	for i := 0; i < 3; i++ {
		if err := enrollments.Create(context.Background(), &model.Enrollment{
			ID:         uuid.New(),
			CourseID:   courseID,
			StudentID:  uuid.New(),
			EnrolledAt: time.Now(),
		}); err != nil {
			t.Fatal(err)
		}
	}

	out, err := uc.Execute(context.Background(), usecase.ListEnrollmentsInput{
		CourseID:   &courseID,
		Pagination: model.Pagination{Page: 1, PageSize: 10},
	})
	if err != nil {
		t.Fatal(err)
	}
	if int(out.TotalCount) != 3 {
		t.Errorf("expected 3 enrollments, got %d", out.TotalCount)
	}
}

func TestListEnrollments_ByStudent(t *testing.T) {
	enrollments := usecasetest.NewFakeEnrollmentRepository()
	uc := usecase.NewListEnrollmentsUseCase(enrollments)

	studentID := uuid.New()
	for i := 0; i < 2; i++ {
		if err := enrollments.Create(context.Background(), &model.Enrollment{
			ID:         uuid.New(),
			CourseID:   uuid.New(),
			StudentID:  studentID,
			EnrolledAt: time.Now(),
		}); err != nil {
			t.Fatal(err)
		}
	}

	out, err := uc.Execute(context.Background(), usecase.ListEnrollmentsInput{
		StudentID:  &studentID,
		Pagination: model.Pagination{Page: 1, PageSize: 10},
	})
	if err != nil {
		t.Fatal(err)
	}
	if int(out.TotalCount) != 2 {
		t.Errorf("expected 2 enrollments for student, got %d", out.TotalCount)
	}
}
