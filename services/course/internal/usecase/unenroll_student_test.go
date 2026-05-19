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

func TestUnenrollStudent_SelfUnenroll(t *testing.T) {
	enrollments := usecasetest.NewFakeEnrollmentRepository()
	courses := usecasetest.NewFakeCourseRepository()
	events := &usecasetest.FakeEventPublisher{}
	enrollUC := usecase.NewEnrollStudentUseCase(enrollments, courses, events, &usecasetest.FakeClock{Fixed: time.Now()})
	uc := usecase.NewUnenrollStudentUseCase(enrollments, courses, events)

	c := seedCourse(t, courses)
	studentID := uuid.New()
	if _, err := enrollUC.Execute(context.Background(), usecase.EnrollStudentInput{
		CourseID: c.ID, StudentID: studentID, CallerID: studentID,
	}); err != nil {
		t.Fatal(err)
	}

	if err := uc.Execute(context.Background(), usecase.UnenrollStudentInput{
		CourseID: c.ID, StudentID: studentID, CallerID: studentID,
	}); err != nil {
		t.Fatal(err)
	}
	if events.LastSubject() != "course.enrollment.removed" {
		t.Errorf("expected 'course.enrollment.removed', got %q", events.LastSubject())
	}
}

func TestUnenrollStudent_PermissionDenied(t *testing.T) {
	enrollments := usecasetest.NewFakeEnrollmentRepository()
	courses := usecasetest.NewFakeCourseRepository()
	events := &usecasetest.FakeEventPublisher{}
	uc := usecase.NewUnenrollStudentUseCase(enrollments, courses, events)

	c := seedCourse(t, courses)
	studentID := uuid.New()
	err := uc.Execute(context.Background(), usecase.UnenrollStudentInput{
		CourseID:  c.ID,
		StudentID: studentID,
		CallerID:  uuid.New(), // neither student nor teacher
	})
	if !isErr(err, model.ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
}
