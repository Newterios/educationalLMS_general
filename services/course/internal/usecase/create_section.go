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

type CreateSectionUseCase struct {
	sections port.SectionRepository
	courses  port.CourseRepository
	cache    port.Cache
	events   port.EventPublisher
}

func NewCreateSectionUseCase(sections port.SectionRepository, courses port.CourseRepository, cache port.Cache, events port.EventPublisher) *CreateSectionUseCase {
	return &CreateSectionUseCase{sections: sections, courses: courses, cache: cache, events: events}
}

type CreateSectionInput struct {
	CourseID uuid.UUID
	CallerID uuid.UUID
	Title    string
	Position int32
}

type CreateSectionOutput struct{ Section *model.Section }

func (uc *CreateSectionUseCase) Execute(ctx context.Context, in CreateSectionInput) (CreateSectionOutput, error) {
	if strings.TrimSpace(in.Title) == "" {
		return CreateSectionOutput{}, fmt.Errorf("section title is required: %w", model.ErrInvalidInput)
	}

	course, err := uc.courses.GetByID(ctx, in.CourseID)
	if err != nil {
		return CreateSectionOutput{}, err
	}
	if course.TeacherID != in.CallerID {
		return CreateSectionOutput{}, fmt.Errorf("create section: %w", model.ErrPermissionDenied)
	}

	s := &model.Section{
		ID:       uuid.Must(uuid.NewV7()),
		CourseID: in.CourseID,
		Title:    strings.TrimSpace(in.Title),
		Position: in.Position,
	}
	if err := uc.sections.Create(ctx, s); err != nil {
		return CreateSectionOutput{}, err
	}

	_ = uc.cache.Delete(ctx, "course:sections:"+in.CourseID.String())

	payload := event.Marshal("course.section.created", s.ID.String(), map[string]string{"course_id": s.CourseID.String()})
	if err := uc.events.Publish(ctx, "course.section.created", payload); err != nil {
		slog.WarnContext(ctx, "publish course.section.created failed", "err", err)
	}

	return CreateSectionOutput{Section: s}, nil
}
