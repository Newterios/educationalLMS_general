package model

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	RefreshHash string
	UserAgent   string
	IP          string
	ExpiresAt   time.Time
	CreatedAt   time.Time
	RevokedAt   *time.Time
}
