package port

import (
	"context"
	"time"

	"github.com/Newterios/lms-system-prob/services/auth/internal/model"
	"github.com/google/uuid"
)

type VerificationCodeRepository interface {
	Create(ctx context.Context, code *model.VerificationCode) error
	GetByCodeHash(ctx context.Context, codeHash string) (*model.VerificationCode, error)
	MarkUsed(ctx context.Context, id uuid.UUID, usedAt time.Time) error
}
