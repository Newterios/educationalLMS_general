package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Newterios/lms-system-prob/services/assessment/internal/model"
	"github.com/Newterios/lms-system-prob/services/assessment/internal/usecase/port"
	"github.com/google/uuid"
)

type CreateQuizUseCase struct {
	quizzes port.QuizRepository
	outbox  port.OutboxRepository
	tx      port.TxManager
}

func NewCreateQuizUseCase(quizzes port.QuizRepository, outbox port.OutboxRepository, tx port.TxManager) *CreateQuizUseCase {
	return &CreateQuizUseCase{quizzes: quizzes, outbox: outbox, tx: tx}
}

type CreateQuizInput struct {
	CourseID     uuid.UUID
	CallerID     uuid.UUID // teacher — denormalized into quiz.teacher_id
	Title        string
	TimeLimitSec int32
	Shuffle      bool
	Questions    []*model.Question
}

type CreateQuizOutput struct{ Quiz *model.Quiz }

func (uc *CreateQuizUseCase) Execute(ctx context.Context, in CreateQuizInput) (CreateQuizOutput, error) {
	if strings.TrimSpace(in.Title) == "" {
		return CreateQuizOutput{}, fmt.Errorf("title required: %w", model.ErrInvalidInput)
	}

	q := &model.Quiz{
		ID:           uuid.Must(uuid.NewRandom()),
		CourseID:     in.CourseID,
		TeacherID:    in.CallerID, // denormalize — avoids cross-service call on future ownership checks
		Title:        strings.TrimSpace(in.Title),
		TimeLimitSec: in.TimeLimitSec,
		Shuffle:      in.Shuffle,
		CreatedAt:    time.Now().UTC(),
		Questions:    in.Questions,
	}

	// assign IDs to questions that don't have them yet
	for _, qs := range q.Questions {
		if qs.ID == uuid.Nil {
			qs.ID = uuid.Must(uuid.NewRandom())
		}
		qs.QuizID = q.ID
	}

	payload, _ := json.Marshal(map[string]string{
		"quiz_id":   q.ID.String(),
		"course_id": q.CourseID.String(),
	})

	var out CreateQuizOutput
	err := uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := uc.quizzes.Create(ctx, q); err != nil {
			return err
		}
		return uc.outbox.Insert(ctx, &model.OutboxEntry{
			AggregateID: q.ID,
			EventType:   "assessment.quiz.created",
			Payload:     payload,
			OccurredAt:  time.Now().UTC(),
		})
	})
	if err != nil {
		return CreateQuizOutput{}, err
	}

	slog.InfoContext(ctx, "CreateQuiz OK", "quiz_id", q.ID, "course_id", q.CourseID)
	out.Quiz = q
	return out, nil
}

const quizCacheTTL = 60 * time.Second

