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

type UnenrollStudentUseCase struct {
	enrollments port.EnrollmentRepository
	courses     port.CourseRepository
	events      port.EventPublisher
}

func NewUnenrollStudentUseCase(enrollments port.EnrollmentRepository, courses port.CourseRepository, events port.EventPublisher) *UnenrollStudentUseCase {
	return &UnenrollStudentUseCase{enrollments: enrollments, courses: courses, events: events}
}

type UnenrollStudentInput struct {
	CourseID  uuid.UUID
	StudentID uuid.UUID
	CallerID  uuid.UUID
}

func (uc *UnenrollStudentUseCase) Execute(ctx context.Context, in UnenrollStudentInput) error {
	// Self-unenroll or teacher/admin via course ownership check.
	if in.CallerID != in.StudentID {
		course, err := uc.courses.GetByID(ctx, in.CourseID)
		if err != nil {
			return err
		}
		if course.TeacherID != in.CallerID {
			return fmt.Errorf("unenroll student: %w", model.ErrPermissionDenied)
		}
	}

	if err := uc.enrollments.Delete(ctx, in.CourseID, in.StudentID); err != nil {
		return err
	}

	payload := event.Marshal("course.enrollment.removed", in.CourseID.String(), map[string]string{"student_id": in.StudentID.String()})
	if err := uc.events.Publish(ctx, "course.enrollment.removed", payload); err != nil {
		slog.WarnContext(ctx, "publish course.enrollment.removed failed", "err", err)
	}

	return nil
}
