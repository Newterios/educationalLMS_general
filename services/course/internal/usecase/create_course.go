package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Newterios/lms-system-prob/services/course/internal/event"
	"github.com/Newterios/lms-system-prob/services/course/internal/model"
	"github.com/Newterios/lms-system-prob/services/course/internal/usecase/port"
	"github.com/google/uuid"
)

type CreateCourseUseCase struct {
	courses port.CourseRepository
	cache   port.Cache
	events  port.EventPublisher
	clock   port.Clock
}

func NewCreateCourseUseCase(courses port.CourseRepository, cache port.Cache, events port.EventPublisher, clock port.Clock) *CreateCourseUseCase {
	return &CreateCourseUseCase{courses: courses, cache: cache, events: events, clock: clock}
}

type CreateCourseInput struct {
	TeacherID   uuid.UUID
	Title       string
	Description string
	Language    string
}

type CreateCourseOutput struct {
	Course *model.Course
}

func (uc *CreateCourseUseCase) Execute(ctx context.Context, in CreateCourseInput) (CreateCourseOutput, error) {
	if strings.TrimSpace(in.Title) == "" {
		return CreateCourseOutput{}, fmt.Errorf("title is required: %w", model.ErrInvalidInput)
	}
	lang := in.Language
	if lang == "" {
		lang = "en"
	}

	now := uc.clock.Now()
	c := &model.Course{
		ID:          uuid.Must(uuid.NewV7()),
		Title:       strings.TrimSpace(in.Title),
		Description: in.Description,
		TeacherID:   in.TeacherID,
		Language:    lang,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := uc.courses.Create(ctx, c); err != nil {
		return CreateCourseOutput{}, err
	}

	_ = uc.cache.DeleteByPrefix(ctx, "course:courses:list:")

	payload := event.Marshal("course.course.created", c.ID.String(), map[string]string{"teacher_id": c.TeacherID.String()})
	if err := uc.events.Publish(ctx, "course.course.created", payload); err != nil {
		slog.WarnContext(ctx, "publish course.course.created failed", "err", err)
	}

	return CreateCourseOutput{Course: c}, nil
}

const courseCacheTTL = 60 * time.Second
