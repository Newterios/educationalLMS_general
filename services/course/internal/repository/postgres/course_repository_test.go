//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/Newterios/lms-system-prob/services/course/internal/model"
	"github.com/Newterios/lms-system-prob/services/course/internal/repository/postgres"
	"github.com/Newterios/lms-system-prob/services/course/internal/usecase/port"
	"github.com/google/uuid"
)

func TestCourseRepository_CreateAndGet(t *testing.T) {
	pool := newTestPool(t)
	repo := postgres.NewCourseRepository(pool)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Millisecond)
	c := &model.Course{
		ID:          uuid.New(),
		Title:       "Integration Test Course",
		Description: "Testing",
		TeacherID:   uuid.New(),
		Language:    "en",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := repo.Create(ctx, c); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Title != c.Title {
		t.Errorf("title mismatch: want %q got %q", c.Title, got.Title)
	}
}

func TestCourseRepository_SoftDelete(t *testing.T) {
	pool := newTestPool(t)
	repo := postgres.NewCourseRepository(pool)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Millisecond)
	c := &model.Course{
		ID: uuid.New(), Title: "To Delete", TeacherID: uuid.New(),
		Language: "en", CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.Create(ctx, c); err != nil {
		t.Fatal(err)
	}
	if err := repo.SoftDelete(ctx, c.ID, now); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}
	_, err := repo.GetByID(ctx, c.ID)
	if err != model.ErrNotFound {
		t.Errorf("expected ErrNotFound after soft delete, got %v", err)
	}
}

func TestCourseRepository_List(t *testing.T) {
	pool := newTestPool(t)
	repo := postgres.NewCourseRepository(pool)
	ctx := context.Background()

	teacherID := uuid.New()
	now := time.Now().UTC().Truncate(time.Millisecond)
	for i := 0; i < 3; i++ {
		c := &model.Course{
			ID: uuid.New(), Title: "Course", TeacherID: teacherID,
			Language: "en", CreatedAt: now, UpdatedAt: now,
		}
		if err := repo.Create(ctx, c); err != nil {
			t.Fatal(err)
		}
	}

	courses, total, err := repo.List(ctx, port.CourseFilter{TeacherID: &teacherID}, model.Pagination{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total < 3 {
		t.Errorf("expected at least 3 courses, got total=%d", total)
	}
	_ = courses
}
