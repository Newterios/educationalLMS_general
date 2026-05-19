package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Newterios/lms-system-prob/services/course/internal/event"
	"github.com/Newterios/lms-system-prob/services/course/internal/model"
	"github.com/Newterios/lms-system-prob/services/course/internal/usecase/port"
	"github.com/google/uuid"
)

type DeleteCourseUseCase struct {
	courses port.CourseRepository
	cache   port.Cache
	events  port.EventPublisher
	clock   port.Clock
}

func NewDeleteCourseUseCase(courses port.CourseRepository, cache port.Cache, events port.EventPublisher, clock port.Clock) *DeleteCourseUseCase {
	return &DeleteCourseUseCase{courses: courses, cache: cache, events: events, clock: clock}
}

type DeleteCourseInput struct {
	ID       uuid.UUID
	CallerID uuid.UUID
}

func (uc *DeleteCourseUseCase) Execute(ctx context.Context, in DeleteCourseInput) error {
	c, err := uc.courses.GetByID(ctx, in.ID)
	if err != nil {
		return err
	}

	if c.TeacherID != in.CallerID {
		return fmt.Errorf("delete course: %w", model.ErrPermissionDenied)
	}

	if err := uc.courses.SoftDelete(ctx, in.ID, uc.clock.Now()); err != nil {
		return err
	}

	_ = uc.cache.Delete(ctx, "course:course:"+in.ID.String())
	_ = uc.cache.DeleteByPrefix(ctx, "course:courses:list:")

	payload := event.Marshal("course.course.deleted", in.ID.String(), nil)
	if err := uc.events.Publish(ctx, "course.course.deleted", payload); err != nil {
		slog.WarnContext(ctx, "publish course.course.deleted failed", "err", err)
	}

	return nil
}
