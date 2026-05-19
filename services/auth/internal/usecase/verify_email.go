package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Newterios/lms-system-prob/services/auth/internal/event"
	"github.com/Newterios/lms-system-prob/services/auth/internal/model"
	"github.com/Newterios/lms-system-prob/services/auth/internal/usecase/port"
)

type VerifyEmailUseCase struct {
	codes  port.VerificationCodeRepository
	users  port.UserRepository
	events port.EventPublisher
	clock  port.Clock
}

func NewVerifyEmailUseCase(
	codes port.VerificationCodeRepository,
	users port.UserRepository,
	events port.EventPublisher,
	clock port.Clock,
) *VerifyEmailUseCase {
	return &VerifyEmailUseCase{codes: codes, users: users, events: events, clock: clock}
}

type VerifyEmailInput struct {
	Code string
}

func (uc *VerifyEmailUseCase) Execute(ctx context.Context, in VerifyEmailInput) error {
	code, err := uc.codes.GetByCodeHash(ctx, sha256Hex(in.Code))
	if err != nil {
		return fmt.Errorf("verify_email: %w: invalid code", model.ErrInvalidInput)
	}

	if code.Kind != "email" {
		return fmt.Errorf("verify_email: %w: wrong code kind", model.ErrInvalidInput)
	}

	now := uc.clock.Now()
	if code.UsedAt != nil {
		return fmt.Errorf("verify_email: %w: code already used", model.ErrFailedPrecondition)
	}
	if now.After(code.ExpiresAt) {
		return fmt.Errorf("verify_email: %w: code expired", model.ErrFailedPrecondition)
	}

	if err := uc.codes.MarkUsed(ctx, code.ID, now); err != nil {
		return fmt.Errorf("verify_email: mark used: %w", err)
	}

	user, err := uc.users.GetByID(ctx, code.UserID)
	if err != nil {
		return fmt.Errorf("verify_email: get user: %w", err)
	}

	user.EmailVerified = true
	user.UpdatedAt = now
	if err := uc.users.Update(ctx, user); err != nil {
		return fmt.Errorf("verify_email: update user: %w", err)
	}

	if err := uc.events.Publish(ctx, "auth.user.verified",
		event.Marshal("auth.user.verified", user.ID.String(), map[string]string{"email": user.Email})); err != nil {
		slog.WarnContext(ctx, "verify_email: publish event failed (best-effort)", "err", err)
	}
	return nil
}
