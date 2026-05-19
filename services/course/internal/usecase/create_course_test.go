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

func TestCreateCourse_HappyPath(t *testing.T) {
	courses := usecasetest.NewFakeCourseRepository()
	cache := usecasetest.NewFakeCache()
	events := &usecasetest.FakeEventPublisher{}
	clock := &usecasetest.FakeClock{Fixed: time.Now()}
	uc := usecase.NewCreateCourseUseCase(courses, cache, events, clock)

	teacherID := uuid.New()
	out, err := uc.Execute(context.Background(), usecase.CreateCourseInput{
		TeacherID:   teacherID,
		Title:       "Go Fundamentals",
		Description: "Learn Go from scratch",
		Language:    "en",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Course.Title != "Go Fundamentals" {
		t.Errorf("expected title 'Go Fundamentals', got %q", out.Course.Title)
	}
	if out.Course.TeacherID != teacherID {
		t.Errorf("expected teacherID %v, got %v", teacherID, out.Course.TeacherID)
	}
	if events.LastSubject() != "course.course.created" {
		t.Errorf("expected event 'course.course.created', got %q", events.LastSubject())
	}
}

func TestCreateCourse_EmptyTitle(t *testing.T) {
	courses := usecasetest.NewFakeCourseRepository()
	cache := usecasetest.NewFakeCache()
	events := &usecasetest.FakeEventPublisher{}
	clock := &usecasetest.FakeClock{Fixed: time.Now()}
	uc := usecase.NewCreateCourseUseCase(courses, cache, events, clock)

	_, err := uc.Execute(context.Background(), usecase.CreateCourseInput{
		TeacherID: uuid.New(),
		Title:     "   ",
	})
	if !isErr(err, model.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateCourse_DefaultLanguage(t *testing.T) {
	courses := usecasetest.NewFakeCourseRepository()
	cache := usecasetest.NewFakeCache()
	events := &usecasetest.FakeEventPublisher{}
	clock := &usecasetest.FakeClock{Fixed: time.Now()}
	uc := usecase.NewCreateCourseUseCase(courses, cache, events, clock)

	out, err := uc.Execute(context.Background(), usecase.CreateCourseInput{
		TeacherID: uuid.New(),
		Title:     "Physics 101",
		Language:  "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Course.Language != "en" {
		t.Errorf("expected default language 'en', got %q", out.Course.Language)
	}
}
