package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Newterios/lms-system-prob/services/auth/internal/event"
	"github.com/Newterios/lms-system-prob/services/auth/internal/model"
	"github.com/Newterios/lms-system-prob/services/auth/internal/usecase/port"
)

type ConfirmPasswordResetUseCase struct {
	codes    port.VerificationCodeRepository
	users    port.UserRepository
	sessions port.SessionRepository
	hasher   port.PasswordHasher
	events   port.EventPublisher
	txRunner port.TxRunner
	clock    port.Clock
}

func NewConfirmPasswordResetUseCase(
	codes port.VerificationCodeRepository,
	users port.UserRepository,
	sessions port.SessionRepository,
	hasher port.PasswordHasher,
	events port.EventPublisher,
	txRunner port.TxRunner,
	clock port.Clock,
) *ConfirmPasswordResetUseCase {
	return &ConfirmPasswordResetUseCase{
		codes: codes, users: users, sessions: sessions,
		hasher: hasher, events: events, txRunner: txRunner, clock: clock,
	}
}

type ConfirmPasswordResetInput struct {
	Code        string
	NewPassword string
}

func (uc *ConfirmPasswordResetUseCase) Execute(ctx context.Context, in ConfirmPasswordResetInput) error {
	if len(in.NewPassword) < 8 {
		return fmt.Errorf("confirm_password_reset: %w: password must be at least 8 characters", model.ErrInvalidInput)
	}

	now := uc.clock.Now()

	code, err := uc.codes.GetByCodeHash(ctx, sha256Hex(in.Code))
	if err != nil {
		return fmt.Errorf("confirm_password_reset: %w: invalid code", model.ErrInvalidInput)
	}

	if code.Kind != "password_reset" {
		return fmt.Errorf("confirm_password_reset: %w: wrong code kind", model.ErrInvalidInput)
	}
	if code.UsedAt != nil {
		return fmt.Errorf("confirm_password_reset: %w: code already used", model.ErrFailedPrecondition)
	}
	if now.After(code.ExpiresAt) {
		return fmt.Errorf("confirm_password_reset: %w: code expired", model.ErrFailedPrecondition)
	}

	newHash, err := uc.hasher.Hash(in.NewPassword)
	if err != nil {
		return fmt.Errorf("confirm_password_reset: hash password: %w", err)
	}

	err = uc.txRunner.WithinTx(ctx, func(ctx context.Context) error {
		if err := uc.codes.MarkUsed(ctx, code.ID, now); err != nil {
			return fmt.Errorf("mark used: %w", err)
		}

		user, err := uc.users.GetByID(ctx, code.UserID)
		if err != nil {
			return fmt.Errorf("get user: %w", err)
		}

		user.PasswordHash = newHash
		user.UpdatedAt = now
		if err := uc.users.Update(ctx, user); err != nil {
			return fmt.Errorf("update user: %w", err)
		}

		if err := uc.sessions.RevokeAllForUser(ctx, user.ID, now); err != nil {
			return fmt.Errorf("revoke sessions: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("confirm_password_reset: %w", err)
	}

	if err := uc.events.Publish(ctx, "auth.password.changed", event.Marshal("auth.password.changed", "user", nil)); err != nil {
		slog.WarnContext(ctx, "confirm_password_reset: publish event failed (best-effort)", "err", err)
	}
	return nil
}
