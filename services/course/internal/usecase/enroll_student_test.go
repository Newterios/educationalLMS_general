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

func TestEnrollStudent_SelfEnroll(t *testing.T) {
	enrollments := usecasetest.NewFakeEnrollmentRepository()
	courses := usecasetest.NewFakeCourseRepository()
	events := &usecasetest.FakeEventPublisher{}
	clock := &usecasetest.FakeClock{Fixed: time.Now()}
	uc := usecase.NewEnrollStudentUseCase(enrollments, courses, events, clock)

	c := seedCourse(t, courses)
	studentID := uuid.New()
	out, err := uc.Execute(context.Background(), usecase.EnrollStudentInput{
		CourseID:  c.ID,
		StudentID: studentID,
		CallerID:  studentID, // self-enroll
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Enrollment.StudentID != studentID {
		t.Errorf("unexpected student ID: %v", out.Enrollment.StudentID)
	}
	if events.LastSubject() != "course.enrollment.created" {
		t.Errorf("expected 'course.enrollment.created', got %q", events.LastSubject())
	}
}

func TestEnrollStudent_TeacherEnrollsStudent(t *testing.T) {
	enrollments := usecasetest.NewFakeEnrollmentRepository()
	courses := usecasetest.NewFakeCourseRepository()
	events := &usecasetest.FakeEventPublisher{}
	clock := &usecasetest.FakeClock{Fixed: time.Now()}
	uc := usecase.NewEnrollStudentUseCase(enrollments, courses, events, clock)

	c := seedCourse(t, courses)
	studentID := uuid.New()
	_, err := uc.Execute(context.Background(), usecase.EnrollStudentInput{
		CourseID:  c.ID,
		StudentID: studentID,
		CallerID:  c.TeacherID, // teacher enrolls
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestEnrollStudent_PermissionDenied(t *testing.T) {
	enrollments := usecasetest.NewFakeEnrollmentRepository()
	courses := usecasetest.NewFakeCourseRepository()
	events := &usecasetest.FakeEventPublisher{}
	clock := &usecasetest.FakeClock{Fixed: time.Now()}
	uc := usecase.NewEnrollStudentUseCase(enrollments, courses, events, clock)

	c := seedCourse(t, courses)
	studentID := uuid.New()
	strangerID := uuid.New()
	_, err := uc.Execute(context.Background(), usecase.EnrollStudentInput{
		CourseID:  c.ID,
		StudentID: studentID,
		CallerID:  strangerID, // neither student nor teacher
	})
	if !isErr(err, model.ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
}

func TestEnrollStudent_DuplicateEnrollment(t *testing.T) {
	enrollments := usecasetest.NewFakeEnrollmentRepository()
	courses := usecasetest.NewFakeCourseRepository()
	events := &usecasetest.FakeEventPublisher{}
	clock := &usecasetest.FakeClock{Fixed: time.Now()}
	uc := usecase.NewEnrollStudentUseCase(enrollments, courses, events, clock)

	c := seedCourse(t, courses)
	studentID := uuid.New()
	in := usecase.EnrollStudentInput{CourseID: c.ID, StudentID: studentID, CallerID: studentID}
	if _, err := uc.Execute(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	_, err := uc.Execute(context.Background(), in)
	if !isErr(err, model.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists on duplicate, got %v", err)
	}
}
