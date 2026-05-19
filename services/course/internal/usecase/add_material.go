package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Newterios/lms-system-prob/services/course/internal/event"
	"github.com/Newterios/lms-system-prob/services/course/internal/model"
	"github.com/Newterios/lms-system-prob/services/course/internal/usecase/port"
	"github.com/google/uuid"
)

type AddMaterialUseCase struct {
	materials port.MaterialRepository
	sections  port.SectionRepository
	courses   port.CourseRepository
	cache     port.Cache
	events    port.EventPublisher
}

func NewAddMaterialUseCase(materials port.MaterialRepository, sections port.SectionRepository, courses port.CourseRepository, cache port.Cache, events port.EventPublisher) *AddMaterialUseCase {
	return &AddMaterialUseCase{materials: materials, sections: sections, courses: courses, cache: cache, events: events}
}

type AddMaterialInput struct {
	SectionID uuid.UUID
	CallerID  uuid.UUID
	Kind      string
	URL       string
	Title     string
}

type AddMaterialOutput struct{ Material *model.Material }

func (uc *AddMaterialUseCase) Execute(ctx context.Context, in AddMaterialInput) (AddMaterialOutput, error) {
	if strings.TrimSpace(in.Title) == "" || strings.TrimSpace(in.URL) == "" {
		return AddMaterialOutput{}, fmt.Errorf("title and url are required: %w", model.ErrInvalidInput)
	}
	kind := in.Kind
	if kind != "pdf" && kind != "video" && kind != "link" {
		return AddMaterialOutput{}, fmt.Errorf("kind must be pdf, video, or link: %w", model.ErrInvalidInput)
	}

	section, err := uc.sections.GetByID(ctx, in.SectionID)
	if err != nil {
		return AddMaterialOutput{}, err
	}

	course, err := uc.courses.GetByID(ctx, section.CourseID)
	if err != nil {
		return AddMaterialOutput{}, err
	}
	if course.TeacherID != in.CallerID {
		return AddMaterialOutput{}, fmt.Errorf("add material: %w", model.ErrPermissionDenied)
	}

	m := &model.Material{
		ID:        uuid.Must(uuid.NewV7()),
		SectionID: in.SectionID,
		Kind:      kind,
		URL:       strings.TrimSpace(in.URL),
		Title:     strings.TrimSpace(in.Title),
	}
	if err := uc.materials.Create(ctx, m); err != nil {
		return AddMaterialOutput{}, err
	}

	_ = uc.cache.Delete(ctx, "course:materials:"+in.SectionID.String())

	payload := event.Marshal("course.material.added", m.ID.String(), map[string]string{"section_id": m.SectionID.String()})
	if err := uc.events.Publish(ctx, "course.material.added", payload); err != nil {
		slog.WarnContext(ctx, "publish course.material.added failed", "err", err)
	}

	return AddMaterialOutput{Material: m}, nil
}
