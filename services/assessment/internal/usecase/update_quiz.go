package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Newterios/lms-system-prob/services/assessment/internal/model"
	"github.com/Newterios/lms-system-prob/services/assessment/internal/usecase/port"
	"github.com/google/uuid"
)

type UpdateQuizUseCase struct {
	quizzes port.QuizRepository
	outbox  port.OutboxRepository
	cache   port.Cache
	tx      port.TxManager
}

func NewUpdateQuizUseCase(quizzes port.QuizRepository, outbox port.OutboxRepository, cache port.Cache, tx port.TxManager) *UpdateQuizUseCase {
	return &UpdateQuizUseCase{quizzes: quizzes, outbox: outbox, cache: cache, tx: tx}
}

type UpdateQuizInput struct {
	ID           uuid.UUID
	CallerID     uuid.UUID
	Title        string
	TimeLimitSec int32
	Shuffle      bool
}

type UpdateQuizOutput struct{ Quiz *model.Quiz }

func (uc *UpdateQuizUseCase) Execute(ctx context.Context, in UpdateQuizInput) (UpdateQuizOutput, error) {
	if strings.TrimSpace(in.Title) == "" {
		return UpdateQuizOutput{}, fmt.Errorf("title required: %w", model.ErrInvalidInput)
	}

	q, err := uc.quizzes.GetByID(ctx, in.ID)
	if err != nil {
		return UpdateQuizOutput{}, err
	}
	// ownership check — uses denormalized teacher_id, no RPC needed
	if q.TeacherID != in.CallerID {
		return UpdateQuizOutput{}, fmt.Errorf("not quiz owner: %w", model.ErrPermissionDenied)
	}

	q.Title = strings.TrimSpace(in.Title)
	q.TimeLimitSec = in.TimeLimitSec
	q.Shuffle = in.Shuffle

	payload, _ := json.Marshal(map[string]string{"quiz_id": q.ID.String()})

	if err := uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := uc.quizzes.Update(ctx, q); err != nil {
			return err
		}
		return uc.outbox.Insert(ctx, &model.OutboxEntry{
			AggregateID: q.ID,
			EventType:   "assessment.quiz.updated",
			Payload:     payload,
			OccurredAt:  time.Now().UTC(),
		})
	}); err != nil {
		return UpdateQuizOutput{}, err
	}

	_ = uc.cache.Delete(ctx, fmt.Sprintf("assessment:quiz:%s", q.ID))
	return UpdateQuizOutput{Quiz: q}, nil
}

