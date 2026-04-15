package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Newterios/lms-system-prob/services/assessment/internal/model"
	"github.com/Newterios/lms-system-prob/services/assessment/internal/usecase/port"
	"github.com/google/uuid"
)

type GetQuizUseCase struct {
	quizzes port.QuizRepository
	cache   port.Cache
}

func NewGetQuizUseCase(quizzes port.QuizRepository, cache port.Cache) *GetQuizUseCase {
	return &GetQuizUseCase{quizzes: quizzes, cache: cache}
}

type GetQuizInput struct{ ID uuid.UUID }
type GetQuizOutput struct{ Quiz *model.Quiz }

func (uc *GetQuizUseCase) Execute(ctx context.Context, in GetQuizInput) (GetQuizOutput, error) {
	key := fmt.Sprintf("assessment:quiz:%s", in.ID)

	if b, err := uc.cache.Get(ctx, key); err == nil && len(b) > 0 {
		var q model.Quiz
		if json.Unmarshal(b, &q) == nil {
			return GetQuizOutput{Quiz: &q}, nil
		}
	}

	q, err := uc.quizzes.GetByID(ctx, in.ID)
	if err != nil {
		return GetQuizOutput{}, err
	}

	if b, err := json.Marshal(q); err == nil {
		_ = uc.cache.Set(ctx, key, b, quizCacheTTL)
	}
	return GetQuizOutput{Quiz: q}, nil
}

