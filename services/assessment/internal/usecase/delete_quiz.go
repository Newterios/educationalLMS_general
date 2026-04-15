package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Newterios/lms-system-prob/services/assessment/internal/model"
	"github.com/Newterios/lms-system-prob/services/assessment/internal/usecase/port"
	"github.com/google/uuid"
)

type DeleteQuizUseCase struct {
	quizzes  port.QuizRepository
	attempts port.AttemptRepository
	outbox   port.OutboxRepository
	cache    port.Cache
	tx       port.TxManager
}

func NewDeleteQuizUseCase(quizzes port.QuizRepository, attempts port.AttemptRepository, outbox port.OutboxRepository, cache port.Cache, tx port.TxManager) *DeleteQuizUseCase {
	return &DeleteQuizUseCase{quizzes: quizzes, attempts: attempts, outbox: outbox, cache: cache, tx: tx}
}

type DeleteQuizInput struct {
	ID       uuid.UUID
	CallerID uuid.UUID
}

func (uc *DeleteQuizUseCase) Execute(ctx context.Context, in DeleteQuizInput) error {
	q, err := uc.quizzes.GetByID(ctx, in.ID)
	if err != nil {
		return err
	}
	if q.TeacherID != in.CallerID {
		return fmt.Errorf("not quiz owner: %w", model.ErrPermissionDenied)
	}

	exists, err := uc.attempts.ExistsForQuiz(ctx, in.ID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("quiz has existing attempts: %w", model.ErrFailedPrecondition)
	}

	payload, _ := json.Marshal(map[string]string{"quiz_id": in.ID.String()})

	if err := uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := uc.quizzes.Delete(ctx, in.ID); err != nil {
			return err
		}
		return uc.outbox.Insert(ctx, &model.OutboxEntry{
			AggregateID: in.ID,
			EventType:   "assessment.quiz.deleted",
			Payload:     payload,
			OccurredAt:  time.Now().UTC(),
		})
	}); err != nil {
		return err
	}

	_ = uc.cache.Delete(ctx, fmt.Sprintf("assessment:quiz:%s", in.ID))
	return nil
}

