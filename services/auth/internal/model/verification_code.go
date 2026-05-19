package model

import (
	"time"

	"github.com/google/uuid"
)

type VerificationCode struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Kind      string // "email" | "password_reset"
	CodeHash  string
	ExpiresAt time.Time
	UsedAt    *time.Time
}
