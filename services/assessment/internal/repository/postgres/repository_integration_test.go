//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/Newterios/lms-system-prob/services/assessment/internal/model"
	"github.com/Newterios/lms-system-prob/services/assessment/internal/repository/postgres"
	"github.com/Newterios/lms-system-prob/services/assessment/internal/usecase/port"
	"github.com/google/uuid"
)

func TestQuizRepository_CreateAndGet(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	repo := postgres.NewQuizRepository(pool)

	quiz := &model.Quiz{
		ID:           uuid.Must(uuid.NewRandom()),
		CourseID:     uuid.Must(uuid.NewRandom()),
		TeacherID:    uuid.Must(uuid.NewRandom()),
		Title:        "Integration Quiz",
		TimeLimitSec: 60,
		Shuffle:      false,
		CreatedAt:    time.Now().UTC().Truncate(time.Millisecond),
		Questions: []*model.Question{
			{
				ID:     uuid.Must(uuid.NewRandom()),
				Body:   "What is 2+2?",
				Points: 1,
				Choices: []*model.Choice{
					{Key: "a", Value: "3", Correct: false},
					{Key: "b", Value: "4", Correct: true},
				},
			},
		},
	}

	if err := repo.Create(ctx, quiz); err != nil {
		t.Fatalf("create quiz: %v", err)
	}

	got, err := repo.GetByID(ctx, quiz.ID)
	if err != nil {
		t.Fatalf("get quiz: %v", err)
	}
	if got.Title != quiz.Title {
		t.Errorf("title mismatch: got %q want %q", got.Title, quiz.Title)
	}
	if len(got.Questions) != 1 {
		t.Fatalf("expected 1 question, got %d", len(got.Questions))
	}
}

func TestAttemptRepository_CRUD(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	quizRepo := postgres.NewQuizRepository(pool)
	attemptRepo := postgres.NewAttemptRepository(pool)

	quiz := &model.Quiz{
		ID: uuid.Must(uuid.NewRandom()), CourseID: uuid.Must(uuid.NewRandom()),
		TeacherID: uuid.Must(uuid.NewRandom()), Title: "Q", TimeLimitSec: 30,
		CreatedAt: time.Now().UTC(),
	}
	if err := quizRepo.Create(ctx, quiz); err != nil {
		t.Fatal(err)
	}

	attempt := &model.Attempt{
		ID:        uuid.Must(uuid.NewRandom()),
		QuizID:    quiz.ID,
		StudentID: uuid.Must(uuid.NewRandom()),
		StartedAt: time.Now().UTC().Truncate(time.Millisecond),
		Status:    "in_progress",
		Answers:   make(map[uuid.UUID][]string),
	}
	if err := attemptRepo.Create(ctx, attempt); err != nil {
		t.Fatalf("create attempt: %v", err)
	}

	got, err := attemptRepo.GetByID(ctx, attempt.ID)
	if err != nil {
		t.Fatalf("get attempt: %v", err)
	}
	if got.Status != "in_progress" {
		t.Errorf("status mismatch: %q", got.Status)
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	score := 100.0
	attempt.SubmittedAt = &now
	attempt.AutoScore = &score
	attempt.Status = "submitted"
	if err := attemptRepo.Update(ctx, attempt); err != nil {
		t.Fatalf("update attempt: %v", err)
	}

	// verify list
	list, total, err := attemptRepo.List(ctx, port.AttemptFilter{QuizID: &quiz.ID}, model.Pagination{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 {
		t.Errorf("expected 1 attempt, got total=%d len=%d", total, len(list))
	}
}

func TestOutboxRepository_InsertListMarkPublished(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()

	repo := postgres.NewOutboxRepository(pool)

	entry := &model.OutboxEntry{
		AggregateID: uuid.Must(uuid.NewRandom()),
		EventType:   "assessment.quiz.created",
		Payload:     []byte(`{"quiz_id":"test"}`),
		OccurredAt:  time.Now().UTC(),
	}

	if err := repo.Insert(ctx, entry); err != nil {
		t.Fatalf("insert outbox: %v", err)
	}

	unpublished, err := repo.ListUnpublished(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(unpublished) == 0 {
		t.Fatal("expected at least 1 unpublished entry")
	}

	entryID := unpublished[0].ID
	if err := repo.MarkPublished(ctx, entryID, time.Now().UTC()); err != nil {
		t.Fatalf("mark published: %v", err)
	}

	afterMark, err := repo.ListUnpublished(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range afterMark {
		if e.ID == entryID {
			t.Error("entry still in unpublished list after MarkPublished")
		}
	}
}
