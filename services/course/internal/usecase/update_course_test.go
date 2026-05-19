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

func TestUpdateCourse_HappyPath(t *testing.T) {
	courses := usecasetest.NewFakeCourseRepository()
	cache := usecasetest.NewFakeCache()
	events := &usecasetest.FakeEventPublisher{}
	clock := &usecasetest.FakeClock{Fixed: time.Now()}
	uc := usecase.NewUpdateCourseUseCase(courses, cache, events, clock)

	c := seedCourse(t, courses)

	out, err := uc.Execute(context.Background(), usecase.UpdateCourseInput{
		ID:       c.ID,
		CallerID: c.TeacherID,
		Title:    "Updated Title",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Course.Title != "Updated Title" {
		t.Errorf("expected 'Updated Title', got %q", out.Course.Title)
	}
	if events.LastSubject() != "course.course.updated" {
		t.Errorf("expected event 'course.course.updated', got %q", events.LastSubject())
	}
}

func TestUpdateCourse_PermissionDenied(t *testing.T) {
	courses := usecasetest.NewFakeCourseRepository()
	cache := usecasetest.NewFakeCache()
	events := &usecasetest.FakeEventPublisher{}
	clock := &usecasetest.FakeClock{Fixed: time.Now()}
	uc := usecase.NewUpdateCourseUseCase(courses, cache, events, clock)

	c := seedCourse(t, courses)

	_, err := uc.Execute(context.Background(), usecase.UpdateCourseInput{
		ID:       c.ID,
		CallerID: uuid.New(), // not the teacher
		Title:    "Hacked Title",
	})
	if !isErr(err, model.ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
}

func TestUpdateCourse_PartialUpdate(t *testing.T) {
	courses := usecasetest.NewFakeCourseRepository()
	cache := usecasetest.NewFakeCache()
	events := &usecasetest.FakeEventPublisher{}
	clock := &usecasetest.FakeClock{Fixed: time.Now()}
	uc := usecase.NewUpdateCourseUseCase(courses, cache, events, clock)

	c := seedCourse(t, courses)
	origTitle := c.Title

	// Update only description — title must remain.
	out, err := uc.Execute(context.Background(), usecase.UpdateCourseInput{
		ID:          c.ID,
		CallerID:    c.TeacherID,
		Description: "New description",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Course.Title != origTitle {
		t.Errorf("title changed unexpectedly: got %q", out.Course.Title)
	}
	if out.Course.Description != "New description" {
		t.Errorf("description not updated: got %q", out.Course.Description)
	}
}
