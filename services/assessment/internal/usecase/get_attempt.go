package usecase

import (
	"context"
	"fmt"

	"github.com/Newterios/lms-system-prob/services/assessment/internal/model"
	"github.com/Newterios/lms-system-prob/services/assessment/internal/usecase/port"
	"github.com/google/uuid"
)

type GetAttemptUseCase struct {
	attempts port.AttemptRepository
	quizzes  port.QuizRepository
}

func NewGetAttemptUseCase(attempts port.AttemptRepository, quizzes port.QuizRepository) *GetAttemptUseCase {
	return &GetAttemptUseCase{attempts: attempts, quizzes: quizzes}
}

type GetAttemptInput struct {
	ID       uuid.UUID
	CallerID uuid.UUID // student-owner or teacher-of-quiz
	Role     string    // "teacher" or "student"
}

type GetAttemptOutput struct{ Attempt *model.Attempt }

func (uc *GetAttemptUseCase) Execute(ctx context.Context, in GetAttemptInput) (GetAttemptOutput, error) {
	attempt, err := uc.attempts.GetByID(ctx, in.ID)
	if err != nil {
		return GetAttemptOutput{}, err
	}

	// student can only see their own attempt
	if in.Role == "student" && attempt.StudentID != in.CallerID {
		return GetAttemptOutput{}, fmt.Errorf("not your attempt: %w", model.ErrPermissionDenied)
	}

	// teacher can see attempt if they own the quiz
	if in.Role == "teacher" {
		quiz, err := uc.quizzes.GetByID(ctx, attempt.QuizID)
		if err != nil {
			return GetAttemptOutput{}, err
		}
		if quiz.TeacherID != in.CallerID {
			return GetAttemptOutput{}, fmt.Errorf("not your quiz: %w", model.ErrPermissionDenied)
		}
	}

	return GetAttemptOutput{Attempt: attempt}, nil
}

