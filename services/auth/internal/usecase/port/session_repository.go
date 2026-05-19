package port

import (
	"context"
	"time"

	"github.com/Newterios/lms-system-prob/services/auth/internal/model"
	"github.com/google/uuid"
)

type SessionRepository interface {
	Create(ctx context.Context, session *model.Session) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Session, error)
	GetByRefreshHash(ctx context.Context, hash string) (*model.Session, error)
	// ListActiveForUser returns sessions that are not revoked and not expired.
	ListActiveForUser(ctx context.Context, userID uuid.UUID) ([]*model.Session, error)
	Revoke(ctx context.Context, id uuid.UUID, revokedAt time.Time) error
	// RevokeAllForUser revokes every session for userID (used after password reset).
	RevokeAllForUser(ctx context.Context, userID uuid.UUID, revokedAt time.Time) error
	// RevokeAllExcept revokes all sessions for userID except the one with keepID
	// (used after ChangePassword to keep the current session alive).
	RevokeAllExcept(ctx context.Context, userID, keepID uuid.UUID, revokedAt time.Time) error
}
