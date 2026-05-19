package model

import "github.com/google/uuid"

type Material struct {
	ID        uuid.UUID
	SectionID uuid.UUID
	Kind      string // "pdf" | "video" | "link"
	URL       string
	Title     string
}
