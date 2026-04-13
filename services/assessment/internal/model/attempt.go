package model

import (
	"time"

	"github.com/google/uuid"
)

type Attempt struct {
	ID          uuid.UUID
	QuizID      uuid.UUID
	StudentID   uuid.UUID
	StartedAt   time.Time
	SubmittedAt *time.Time
	AutoScore   *float64
	ManualScore *float64
	Status      string // "in_progress" | "submitted" | "graded"
	// Answers maps QuestionID → list of chosen ChoiceKeys
	Answers map[uuid.UUID][]string
}
