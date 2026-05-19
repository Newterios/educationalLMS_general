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

func TestGetCourse_HappyPath(t *testing.T) {
	courses := usecasetest.NewFakeCourseRepository()
	cache := usecasetest.NewFakeCache()
	uc := usecase.NewGetCourseUseCase(courses, cache)

	c := seedCourse(t, courses)
	out, err := uc.Execute(context.Background(), usecase.GetCourseInput{ID: c.ID})
	if err != nil {
		t.Fatal(err)
	}
	if out.Course.ID != c.ID {
		t.Errorf("expected course ID %v, got %v", c.ID, out.Course.ID)
	}
}

func TestGetCourse_NotFound(t *testing.T) {
	courses := usecasetest.NewFakeCourseRepository()
	cache := usecasetest.NewFakeCache()
	uc := usecase.NewGetCourseUseCase(courses, cache)

	_, err := uc.Execute(context.Background(), usecase.GetCourseInput{ID: uuid.New()})
	if !isErr(err, model.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetCourse_ServedFromCache(t *testing.T) {
	courses := usecasetest.NewFakeCourseRepository()
	cache := usecasetest.NewFakeCache()
	uc := usecase.NewGetCourseUseCase(courses, cache)

	c := seedCourse(t, courses)

	// First call populates cache.
	if _, err := uc.Execute(context.Background(), usecase.GetCourseInput{ID: c.ID}); err != nil {
		t.Fatal(err)
	}

	// Simulate DB unavailability by forcing error.
	courses.ForceErr = model.ErrNotFound

	// Second call must still succeed via cache.
	out, err := uc.Execute(context.Background(), usecase.GetCourseInput{ID: c.ID})
	if err != nil {
		t.Fatalf("expected cache hit, got error: %v", err)
	}
	if out.Course.ID != c.ID {
		t.Error("cache returned wrong course")
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func seedCourse(t *testing.T, repo *usecasetest.FakeCourseRepository) *model.Course {
	t.Helper()
	now := time.Now()
	c := &model.Course{
		ID:        uuid.New(),
		Title:     "Test Course",
		TeacherID: uuid.New(),
		Language:  "en",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.Create(context.Background(), c); err != nil {
		t.Fatalf("seedCourse: %v", err)
	}
	return c
}
