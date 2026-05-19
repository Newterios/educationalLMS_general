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

func TestDeleteCourse_HappyPath(t *testing.T) {
	courses := usecasetest.NewFakeCourseRepository()
	cache := usecasetest.NewFakeCache()
	events := &usecasetest.FakeEventPublisher{}
	clock := &usecasetest.FakeClock{Fixed: time.Now()}
	uc := usecase.NewDeleteCourseUseCase(courses, cache, events, clock)

	c := seedCourse(t, courses)
	if err := uc.Execute(context.Background(), usecase.DeleteCourseInput{ID: c.ID, CallerID: c.TeacherID}); err != nil {
		t.Fatal(err)
	}
	if events.LastSubject() != "course.course.deleted" {
		t.Errorf("expected event 'course.course.deleted', got %q", events.LastSubject())
	}

	// Should be soft-deleted: GetByID must return NotFound.
	_, err := courses.GetByID(context.Background(), c.ID)
	if !isErr(err, model.ErrNotFound) {
		t.Errorf("expected soft-deleted course to be not found, got %v", err)
	}
}

func TestDeleteCourse_PermissionDenied(t *testing.T) {
	courses := usecasetest.NewFakeCourseRepository()
	cache := usecasetest.NewFakeCache()
	events := &usecasetest.FakeEventPublisher{}
	clock := &usecasetest.FakeClock{Fixed: time.Now()}
	uc := usecase.NewDeleteCourseUseCase(courses, cache, events, clock)

	c := seedCourse(t, courses)
	err := uc.Execute(context.Background(), usecase.DeleteCourseInput{ID: c.ID, CallerID: uuid.New()})
	if !isErr(err, model.ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
}
