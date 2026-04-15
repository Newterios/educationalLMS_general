package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Newterios/lms-system-prob/services/assessment/internal/model"
	"github.com/Newterios/lms-system-prob/services/assessment/internal/usecase/port"
	"github.com/google/uuid"
)

type ListQuizzesUseCase struct {
	quizzes port.QuizRepository
	cache   port.Cache
}

func NewListQuizzesUseCase(quizzes port.QuizRepository, cache port.Cache) *ListQuizzesUseCase {
	return &ListQuizzesUseCase{quizzes: quizzes, cache: cache}
}

type ListQuizzesInput struct {
	CourseID   uuid.UUID
	Pagination model.Pagination
}

type ListQuizzesOutput struct {
	Quizzes    []*model.Quiz
	TotalCount int64
}

func (uc *ListQuizzesUseCase) Execute(ctx context.Context, in ListQuizzesInput) (ListQuizzesOutput, error) {
	type cached struct {
		Quizzes    []*model.Quiz `json:"quizzes"`
		TotalCount int64         `json:"total"`
	}

	key := fmt.Sprintf("assessment:quizzes:%s", in.CourseID)
	if b, err := uc.cache.Get(ctx, key); err == nil && len(b) > 0 {
		var c cached
		if json.Unmarshal(b, &c) == nil {
			return ListQuizzesOutput{Quizzes: c.Quizzes, TotalCount: c.TotalCount}, nil
		}
	}

	quizzes, total, err := uc.quizzes.ListByCourseID(ctx, in.CourseID, in.Pagination)
	if err != nil {
		return ListQuizzesOutput{}, err
	}

	if b, err := json.Marshal(cached{Quizzes: quizzes, TotalCount: total}); err == nil {
		_ = uc.cache.Set(ctx, key, b, quizCacheTTL)
	}
	return ListQuizzesOutput{Quizzes: quizzes, TotalCount: total}, nil
}

