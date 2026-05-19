package model

import "github.com/google/uuid"

type Section struct {
	ID       uuid.UUID
	CourseID uuid.UUID
	Title    string
	Position int32
}
