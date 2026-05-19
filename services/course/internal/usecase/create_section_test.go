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

func TestCreateSection_HappyPath(t *testing.T) {
	courses := usecasetest.NewFakeCourseRepository()
	sections := usecasetest.NewFakeSectionRepository()
	cache := usecasetest.NewFakeCache()
	events := &usecasetest.FakeEventPublisher{}
	uc := usecase.NewCreateSectionUseCase(sections, courses, cache, events)

	c := seedCourse(t, courses)
	out, err := uc.Execute(context.Background(), usecase.CreateSectionInput{
		CourseID: c.ID,
		CallerID: c.TeacherID,
		Title:    "Introduction",
		Position: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Section.Title != "Introduction" {
		t.Errorf("expected 'Introduction', got %q", out.Section.Title)
	}
	if events.LastSubject() != "course.section.created" {
		t.Errorf("expected 'course.section.created', got %q", events.LastSubject())
	}
}

func TestCreateSection_PermissionDenied(t *testing.T) {
	courses := usecasetest.NewFakeCourseRepository()
	sections := usecasetest.NewFakeSectionRepository()
	cache := usecasetest.NewFakeCache()
	events := &usecasetest.FakeEventPublisher{}
	uc := usecase.NewCreateSectionUseCase(sections, courses, cache, events)

	c := seedCourse(t, courses)
	_, err := uc.Execute(context.Background(), usecase.CreateSectionInput{
		CourseID: c.ID,
		CallerID: uuid.New(),
		Title:    "Hacked Section",
	})
	if !isErr(err, model.ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
}

func TestCreateSection_EmptyTitle(t *testing.T) {
	courses := usecasetest.NewFakeCourseRepository()
	sections := usecasetest.NewFakeSectionRepository()
	cache := usecasetest.NewFakeCache()
	events := &usecasetest.FakeEventPublisher{}
	uc := usecase.NewCreateSectionUseCase(sections, courses, cache, events)

	c := seedCourse(t, courses)
	_, err := uc.Execute(context.Background(), usecase.CreateSectionInput{
		CourseID: c.ID,
		CallerID: c.TeacherID,
		Title:    "",
	})
	if !isErr(err, model.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ── shared helpers (visible to all files in package usecase_test) ─────────────

func isErr(err, target error) bool {
	if err == nil {
		return false
	}
	return containsTarget(err, target)
}

func containsTarget(err, target error) bool {
	if err == target {
		return true
	}
	type unwrapper interface{ Unwrap() error }
	if uw, ok := err.(unwrapper); ok {
		return containsTarget(uw.Unwrap(), target)
	}
	return false
}

func seedCourseWithTeacher(t *testing.T, repo *usecasetest.FakeCourseRepository, teacherID uuid.UUID) *model.Course {
	t.Helper()
	now := time.Now()
	c := &model.Course{
		ID:        uuid.New(),
		Title:     "Test Course",
		TeacherID: teacherID,
		Language:  "en",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.Create(context.Background(), c); err != nil {
		t.Fatalf("seedCourseWithTeacher: %v", err)
	}
	return c
}
