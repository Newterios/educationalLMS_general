package usecase_test

import (
	"context"
	"testing"

	"github.com/Newterios/lms-system-prob/services/course/internal/model"
	"github.com/Newterios/lms-system-prob/services/course/internal/usecase"
	"github.com/Newterios/lms-system-prob/services/course/internal/usecase/usecasetest"
	"github.com/google/uuid"
)

func TestAddMaterial_HappyPath(t *testing.T) {
	materials := usecasetest.NewFakeMaterialRepository()
	sections := usecasetest.NewFakeSectionRepository()
	courses := usecasetest.NewFakeCourseRepository()
	cache := usecasetest.NewFakeCache()
	events := &usecasetest.FakeEventPublisher{}
	uc := usecase.NewAddMaterialUseCase(materials, sections, courses, cache, events)

	c := seedCourse(t, courses)
	section := seedSection(t, sections, c.ID)

	out, err := uc.Execute(context.Background(), usecase.AddMaterialInput{
		SectionID: section.ID,
		CallerID:  c.TeacherID,
		Kind:      "pdf",
		URL:       "https://example.com/doc.pdf",
		Title:     "Lecture Notes",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Material.Kind != "pdf" {
		t.Errorf("expected kind 'pdf', got %q", out.Material.Kind)
	}
	if events.LastSubject() != "course.material.added" {
		t.Errorf("expected 'course.material.added', got %q", events.LastSubject())
	}
}

func TestAddMaterial_InvalidKind(t *testing.T) {
	materials := usecasetest.NewFakeMaterialRepository()
	sections := usecasetest.NewFakeSectionRepository()
	courses := usecasetest.NewFakeCourseRepository()
	cache := usecasetest.NewFakeCache()
	events := &usecasetest.FakeEventPublisher{}
	uc := usecase.NewAddMaterialUseCase(materials, sections, courses, cache, events)

	c := seedCourse(t, courses)
	section := seedSection(t, sections, c.ID)

	_, err := uc.Execute(context.Background(), usecase.AddMaterialInput{
		SectionID: section.ID,
		CallerID:  c.TeacherID,
		Kind:      "image",
		URL:       "https://example.com/img.jpg",
		Title:     "Banner",
	})
	if !isErr(err, model.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for unsupported kind, got %v", err)
	}
}

func TestAddMaterial_PermissionDenied(t *testing.T) {
	materials := usecasetest.NewFakeMaterialRepository()
	sections := usecasetest.NewFakeSectionRepository()
	courses := usecasetest.NewFakeCourseRepository()
	cache := usecasetest.NewFakeCache()
	events := &usecasetest.FakeEventPublisher{}
	uc := usecase.NewAddMaterialUseCase(materials, sections, courses, cache, events)

	c := seedCourse(t, courses)
	section := seedSection(t, sections, c.ID)

	_, err := uc.Execute(context.Background(), usecase.AddMaterialInput{
		SectionID: section.ID,
		CallerID:  uuid.New(),
		Kind:      "link",
		URL:       "https://example.com",
		Title:     "External Link",
	})
	if !isErr(err, model.ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
}

func seedSection(t *testing.T, repo *usecasetest.FakeSectionRepository, courseID uuid.UUID) *model.Section {
	t.Helper()
	s := &model.Section{
		ID:       uuid.New(),
		CourseID: courseID,
		Title:    "Test Section",
		Position: 1,
	}
	if err := repo.Create(context.Background(), s); err != nil {
		t.Fatalf("seedSection: %v", err)
	}
	return s
}
