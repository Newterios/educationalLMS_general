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

func TestListCourses_AllCourses(t *testing.T) {
	courses := usecasetest.NewFakeCourseRepository()
	cache := usecasetest.NewFakeCache()
	uc := usecase.NewListCoursesUseCase(courses, cache)

	for i := 0; i < 3; i++ {
		now := time.Now()
		c := &model.Course{
			ID:        uuid.New(),
			Title:     "Course",
			TeacherID: uuid.New(),
			Language:  "en",
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := courses.Create(context.Background(), c); err != nil {
			t.Fatal(err)
		}
	}

	out, err := uc.Execute(context.Background(), usecase.ListCoursesInput{
		Pagination: model.Pagination{Page: 1, PageSize: 10},
	})
	if err != nil {
		t.Fatal(err)
	}
	if int(out.TotalCount) < 3 {
		t.Errorf("expected at least 3 courses, got %d", out.TotalCount)
	}
}

func TestListCourses_FilterByTeacher(t *testing.T) {
	courses := usecasetest.NewFakeCourseRepository()
	cache := usecasetest.NewFakeCache()
	uc := usecase.NewListCoursesUseCase(courses, cache)

	teacherID := uuid.New()
	for i := 0; i < 2; i++ {
		now := time.Now()
		c := &model.Course{
			ID: uuid.New(), Title: "Mine", TeacherID: teacherID,
			Language: "en", CreatedAt: now, UpdatedAt: now,
		}
		if err := courses.Create(context.Background(), c); err != nil {
			t.Fatal(err)
		}
	}
	// Add a course for a different teacher.
	other := uuid.New()
	now := time.Now()
	if err := courses.Create(context.Background(), &model.Course{
		ID: uuid.New(), Title: "Other", TeacherID: other,
		Language: "en", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	out, err := uc.Execute(context.Background(), usecase.ListCoursesInput{
		TeacherID:  &teacherID,
		Pagination: model.Pagination{Page: 1, PageSize: 10},
	})
	if err != nil {
		t.Fatal(err)
	}
	if int(out.TotalCount) != 2 {
		t.Errorf("expected 2 courses for teacher, got %d", out.TotalCount)
	}
}

func TestListCourses_ServedFromCache(t *testing.T) {
	courses := usecasetest.NewFakeCourseRepository()
	cache := usecasetest.NewFakeCache()
	uc := usecase.NewListCoursesUseCase(courses, cache)

	in := usecase.ListCoursesInput{Pagination: model.Pagination{Page: 1, PageSize: 10}}
	// Prime cache.
	if _, err := uc.Execute(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	// Break DB.
	courses.ForceErr = model.ErrNotFound
	// Must still succeed from cache.
	if _, err := uc.Execute(context.Background(), in); err != nil {
		t.Fatalf("expected cache hit, got %v", err)
	}
}
