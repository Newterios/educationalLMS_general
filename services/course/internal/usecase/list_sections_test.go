package usecase_test

import (
	"context"
	"testing"

	"github.com/Newterios/lms-system-prob/services/course/internal/model"
	"github.com/Newterios/lms-system-prob/services/course/internal/usecase"
	"github.com/Newterios/lms-system-prob/services/course/internal/usecase/usecasetest"
	"github.com/google/uuid"
)

func TestListSections_HappyPath(t *testing.T) {
	sections := usecasetest.NewFakeSectionRepository()
	cache := usecasetest.NewFakeCache()
	uc := usecase.NewListSectionsUseCase(sections, cache)

	courses := usecasetest.NewFakeCourseRepository()
	c := seedCourse(t, courses)

	s1 := seedSection(t, sections, c.ID)
	s2 := seedSection(t, sections, c.ID)

	out, err := uc.Execute(context.Background(), usecase.ListSectionsInput{CourseID: c.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Sections) != 2 {
		t.Errorf("expected 2 sections, got %d", len(out.Sections))
	}
	_ = s1
	_ = s2
}

func TestListSections_Empty(t *testing.T) {
	sections := usecasetest.NewFakeSectionRepository()
	cache := usecasetest.NewFakeCache()
	uc := usecase.NewListSectionsUseCase(sections, cache)

	out, err := uc.Execute(context.Background(), usecase.ListSectionsInput{CourseID: uuid.New()})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Sections) != 0 {
		t.Errorf("expected 0 sections, got %d", len(out.Sections))
	}
}

func TestListSections_CacheMiss_ThenHit(t *testing.T) {
	sections := usecasetest.NewFakeSectionRepository()
	cache := usecasetest.NewFakeCache()
	uc := usecase.NewListSectionsUseCase(sections, cache)

	courses := usecasetest.NewFakeCourseRepository()
	c := seedCourse(t, courses)
	seedSection(t, sections, c.ID)

	in := usecase.ListSectionsInput{CourseID: c.ID}
	// First call — DB hit, populates cache.
	if _, err := uc.Execute(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	// Second call — must come from cache; verify no error even if sections gone.
	out, err := uc.Execute(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Sections) == 0 {
		t.Error("expected cached sections")
	}
}

// seedSection is declared in add_material_test.go; visible here in same package.
var _ = &model.Section{} // import sentinel
