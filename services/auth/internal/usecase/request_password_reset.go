package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Newterios/lms-system-prob/services/auth/internal/event"
	"github.com/Newterios/lms-system-prob/services/auth/internal/model"
	"github.com/Newterios/lms-system-prob/services/auth/internal/usecase/port"
	"github.com/google/uuid"
)

type RequestPasswordResetUseCase struct {
	users   port.UserRepository
	codes   port.VerificationCodeRepository
	events  port.EventPublisher
	mailer  port.Mailer
	codeGen port.CodeGenerator
	clock   port.Clock
}

func NewRequestPasswordResetUseCase(
	users port.UserRepository,
	codes port.VerificationCodeRepository,
	events port.EventPublisher,
	mailer port.Mailer,
	codeGen port.CodeGenerator,
	clock port.Clock,
) *RequestPasswordResetUseCase {
	return &RequestPasswordResetUseCase{users: users, codes: codes, events: events, mailer: mailer, codeGen: codeGen, clock: clock}
}

type RequestPasswordResetInput struct {
	Email string
}

func (uc *RequestPasswordResetUseCase) Execute(ctx context.Context, in RequestPasswordResetInput) error {
	email := strings.ToLower(strings.TrimSpace(in.Email))

	user, err := uc.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil // anti-enumeration: always succeed
		}
		return fmt.Errorf("request_password_reset: get user: %w", err)
	}

	raw, codeHash, err := uc.codeGen.Generate()
	if err != nil {
		return fmt.Errorf("request_password_reset: generate code: %w", err)
	}

	codeID, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("request_password_reset: new code id: %w", err)
	}

	now := uc.clock.Now()
	code := &model.VerificationCode{
		ID:        codeID,
		UserID:    user.ID,
		Kind:      "password_reset",
		CodeHash:  codeHash,
		ExpiresAt: now.Add(1 * time.Hour),
	}
	if err := uc.codes.Create(ctx, code); err != nil {
		return fmt.Errorf("request_password_reset: create code: %w", err)
	}

	if err := uc.mailer.SendPasswordResetEmail(ctx, email, user.FullName, raw); err != nil {
		slog.WarnContext(ctx, "request_password_reset: send email failed (best-effort)", "err", err)
	}
	if err := uc.events.Publish(ctx, "auth.password.reset_requested", event.Marshal("auth.password.reset_requested", "user", nil)); err != nil {
		slog.WarnContext(ctx, "request_password_reset: publish event failed (best-effort)", "err", err)
	}
	return nil
}
