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

type UpdateCourseUseCase struct {
	courses port.CourseRepository
	cache   port.Cache
	events  port.EventPublisher
	clock   port.Clock
}

func NewUpdateCourseUseCase(courses port.CourseRepository, cache port.Cache, events port.EventPublisher, clock port.Clock) *UpdateCourseUseCase {
	return &UpdateCourseUseCase{courses: courses, cache: cache, events: events, clock: clock}
}

type UpdateCourseInput struct {
	ID          uuid.UUID
	CallerID    uuid.UUID
	Title       string
	Description string
	Language    string
}

type UpdateCourseOutput struct{ Course *model.Course }

func (uc *UpdateCourseUseCase) Execute(ctx context.Context, in UpdateCourseInput) (UpdateCourseOutput, error) {
	c, err := uc.courses.GetByID(ctx, in.ID)
	if err != nil {
		return UpdateCourseOutput{}, err
	}

	if c.TeacherID != in.CallerID {
		return UpdateCourseOutput{}, fmt.Errorf("update course: %w", model.ErrPermissionDenied)
	}

	if t := strings.TrimSpace(in.Title); t != "" {
		c.Title = t
	}
	if in.Description != "" {
		c.Description = in.Description
	}
	if in.Language != "" {
		c.Language = in.Language
	}
	c.UpdatedAt = uc.clock.Now()

	if err := uc.courses.Update(ctx, c); err != nil {
		return UpdateCourseOutput{}, err
	}

	_ = uc.cache.Delete(ctx, "course:course:"+c.ID.String())
	_ = uc.cache.DeleteByPrefix(ctx, "course:courses:list:")

	payload := event.Marshal("course.course.updated", c.ID.String(), nil)
	if err := uc.events.Publish(ctx, "course.course.updated", payload); err != nil {
		slog.WarnContext(ctx, "publish course.course.updated failed", "err", err)
	}

	return UpdateCourseOutput{Course: c}, nil
}
