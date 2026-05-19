package model

import (
	"time"

	"github.com/google/uuid"
)

type Course struct {
	ID          uuid.UUID
	Title       string
	Description string
	TeacherID   uuid.UUID
	Language    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}
