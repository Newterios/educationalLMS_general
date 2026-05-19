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

type EnrollStudentUseCase struct {
	enrollments port.EnrollmentRepository
	courses     port.CourseRepository
	events      port.EventPublisher
	clock       port.Clock
}

func NewEnrollStudentUseCase(enrollments port.EnrollmentRepository, courses port.CourseRepository, events port.EventPublisher, clock port.Clock) *EnrollStudentUseCase {
	return &EnrollStudentUseCase{enrollments: enrollments, courses: courses, events: events, clock: clock}
}

type EnrollStudentInput struct {
	CourseID  uuid.UUID
	StudentID uuid.UUID
	CallerID  uuid.UUID
}

type EnrollStudentOutput struct{ Enrollment *model.Enrollment }

func (uc *EnrollStudentUseCase) Execute(ctx context.Context, in EnrollStudentInput) (EnrollStudentOutput, error) {
	// Self-enroll or teacher/admin enroll via course ownership check.
	if in.CallerID != in.StudentID {
		course, err := uc.courses.GetByID(ctx, in.CourseID)
		if err != nil {
			return EnrollStudentOutput{}, err
		}
		if course.TeacherID != in.CallerID {
			return EnrollStudentOutput{}, fmt.Errorf("enroll student: %w", model.ErrPermissionDenied)
		}
	}

	e := &model.Enrollment{
		ID:         uuid.Must(uuid.NewV7()),
		CourseID:   in.CourseID,
		StudentID:  in.StudentID,
		EnrolledAt: uc.clock.Now(),
	}
	if err := uc.enrollments.Create(ctx, e); err != nil {
		return EnrollStudentOutput{}, err
	}

	payload := event.Marshal("course.enrollment.created", e.ID.String(), map[string]string{"course_id": e.CourseID.String(), "student_id": e.StudentID.String()})
	if err := uc.events.Publish(ctx, "course.enrollment.created", payload); err != nil {
		slog.WarnContext(ctx, "publish course.enrollment.created failed", "err", err)
	}

	return EnrollStudentOutput{Enrollment: e}, nil
}
