package model

import (
	"time"

	"github.com/google/uuid"
)

type Enrollment struct {
	ID         uuid.UUID
	CourseID   uuid.UUID
	StudentID  uuid.UUID
	EnrolledAt time.Time
}
