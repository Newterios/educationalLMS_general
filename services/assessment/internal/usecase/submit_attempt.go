package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/Newterios/lms-system-prob/services/assessment/internal/event"
	"github.com/Newterios/lms-system-prob/services/assessment/internal/model"
	"github.com/Newterios/lms-system-prob/services/assessment/internal/usecase/port"
	"github.com/google/uuid"
)

// SubmitAttemptUseCase runs auto-grading inside a DB transaction and writes
// the result row + an outbox entry atomically (ARCHITECTURE.md §4.3).
type SubmitAttemptUseCase struct {
	quizzes  port.QuizRepository
	attempts port.AttemptRepository
	outbox   port.OutboxRepository
	tx       port.TxManager
}

func NewSubmitAttemptUseCase(quizzes port.QuizRepository, attempts port.AttemptRepository, outbox port.OutboxRepository, tx port.TxManager) *SubmitAttemptUseCase {
	return &SubmitAttemptUseCase{quizzes: quizzes, attempts: attempts, outbox: outbox, tx: tx}
}

type SubmitAttemptInput struct {
	AttemptID uuid.UUID
	StudentID uuid.UUID              // ownership check
	Answers   map[uuid.UUID][]string // questionID → []choiceKey
}

type SubmitAttemptOutput struct{ Attempt *model.Attempt }

func (uc *SubmitAttemptUseCase) Execute(ctx context.Context, in SubmitAttemptInput) (SubmitAttemptOutput, error) {
	// load attempt outside TX (read-only)
	attempt, err := uc.attempts.GetByID(ctx, in.AttemptID)
	if err != nil {
		return SubmitAttemptOutput{}, err
	}
	if attempt.StudentID != in.StudentID {
		return SubmitAttemptOutput{}, fmt.Errorf("not attempt owner: %w", model.ErrPermissionDenied)
	}
	if attempt.Status != "in_progress" {
		return SubmitAttemptOutput{}, fmt.Errorf("attempt already submitted: %w", model.ErrFailedPrecondition)
	}

	// load quiz to get correct answers for auto-grading
	quiz, err := uc.quizzes.GetByID(ctx, attempt.QuizID)
	if err != nil {
		return SubmitAttemptOutput{}, err
	}

	// auto-grade: sum points for correct answers
	autoScore := autoGrade(quiz.Questions, in.Answers)

	now := time.Now().UTC()
	attempt.SubmittedAt = &now
	attempt.AutoScore = &autoScore
	attempt.Status = "submitted"
	attempt.Answers = in.Answers

	payload := event.Marshal("assessment.attempt.submitted", attempt.ID.String(), map[string]any{
		"quiz_id":    attempt.QuizID.String(),
		"student_id": attempt.StudentID.String(),
		"auto_score": autoScore,
	})

	// ACID: update attempt + insert outbox in one TX.
	// If outbox.Insert fails, attempt.Update is rolled back too.
	if err := uc.tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := uc.attempts.Update(ctx, attempt); err != nil {
			return err
		}
		return uc.outbox.Insert(ctx, &model.OutboxEntry{
			AggregateID: attempt.ID,
			EventType:   "assessment.attempt.submitted",
			Payload:     payload,
			OccurredAt:  time.Now().UTC(),
		})
	}); err != nil {
		return SubmitAttemptOutput{}, err
	}

	return SubmitAttemptOutput{Attempt: attempt}, nil
}

// autoGrade computes the percentage of total possible points the student earned.
// Returns a value in [0, 100].
func autoGrade(questions []*model.Question, answers map[uuid.UUID][]string) float64 {
	var totalPoints, earnedPoints float64
	for _, q := range questions {
		totalPoints += float64(q.Points)
		chosen := answers[q.ID]
		if allCorrect(q.Choices, chosen) {
			earnedPoints += float64(q.Points)
		}
	}
	if totalPoints == 0 {
		return 0
	}
	return (earnedPoints / totalPoints) * 100
}

// allCorrect checks that exactly the correct choices were selected.
func allCorrect(choices []*model.Choice, chosen []string) bool {
	correctKeys := map[string]bool{}
	for _, c := range choices {
		if c.Correct {
			correctKeys[c.Key] = true
		}
	}
	if len(chosen) != len(correctKeys) {
		return false
	}
	for _, k := range chosen {
		if !correctKeys[k] {
			return false
		}
	}
	return true
}

