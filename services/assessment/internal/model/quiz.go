package model

import (
	"time"

	"github.com/google/uuid"
)

type Quiz struct {
	ID           uuid.UUID
	CourseID     uuid.UUID
	// TeacherID is denormalized from the JWT at CreateQuiz time to avoid
	// cross-service lookups on every UpdateQuiz / DeleteQuiz ownership check.
	TeacherID    uuid.UUID
	Title        string
	TimeLimitSec int32
	Shuffle      bool
	CreatedAt    time.Time
	Questions    []*Question
}

type Question struct {
	ID      uuid.UUID
	QuizID  uuid.UUID
	Body    string
	Choices []*Choice
	Points  int32
}

// Choice is the internal model used for storage and auto-grading.
// Correct is present here; it is stripped by the mapper before any proto response.
type Choice struct {
	Key     string
	Value   string
	Correct bool
}
