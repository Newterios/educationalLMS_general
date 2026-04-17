package usecase

import (
	"context"

	"github.com/Newterios/lms-system-prob/services/assessment/internal/model"
	"github.com/Newterios/lms-system-prob/services/assessment/internal/usecase/port"
	"github.com/google/uuid"
)

type ListAttemptsUseCase struct {
	attempts port.AttemptRepository
}

func NewListAttemptsUseCase(attempts port.AttemptRepository) *ListAttemptsUseCase {
	return &ListAttemptsUseCase{attempts: attempts}
}

type ListAttemptsInput struct {
	QuizID     *uuid.UUID
	StudentID  *uuid.UUID
	Pagination model.Pagination
}

type ListAttemptsOutput struct {
	Attempts   []*model.Attempt
	TotalCount int64
}

func (uc *ListAttemptsUseCase) Execute(ctx context.Context, in ListAttemptsInput) (ListAttemptsOutput, error) {
	attempts, total, err := uc.attempts.List(ctx, port.AttemptFilter{
		QuizID:    in.QuizID,
		StudentID: in.StudentID,
	}, in.Pagination)
	if err != nil {
		return ListAttemptsOutput{}, err
	}
	return ListAttemptsOutput{Attempts: attempts, TotalCount: total}, nil
}

