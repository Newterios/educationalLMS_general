package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Newterios/lms-system-prob/services/auth/internal/event"
	"github.com/Newterios/lms-system-prob/services/auth/internal/model"
	"github.com/Newterios/lms-system-prob/services/auth/internal/usecase/port"
	"github.com/google/uuid"
)

type ChangePasswordUseCase struct {
	users    port.UserRepository
	sessions port.SessionRepository
	hasher   port.PasswordHasher
	events   port.EventPublisher
	txRunner port.TxRunner
	clock    port.Clock
}

func NewChangePasswordUseCase(
	users port.UserRepository,
	sessions port.SessionRepository,
	hasher port.PasswordHasher,
	events port.EventPublisher,
	txRunner port.TxRunner,
	clock port.Clock,
) *ChangePasswordUseCase {
	return &ChangePasswordUseCase{
		users: users, sessions: sessions, hasher: hasher,
		events: events, txRunner: txRunner, clock: clock,
	}
}

type ChangePasswordInput struct {
	UserID          uuid.UUID // taken from JWT in the gRPC handler
	SessionID       uuid.UUID // current session — kept alive after password change
	OldPassword     string
	NewPassword     string
}

func (uc *ChangePasswordUseCase) Execute(ctx context.Context, in ChangePasswordInput) error {
	if len(in.NewPassword) < 8 {
		return fmt.Errorf("change_password: %w: password must be at least 8 characters", model.ErrInvalidInput)
	}

	user, err := uc.users.GetByID(ctx, in.UserID)
	if err != nil {
		return fmt.Errorf("change_password: get user: %w", err)
	}

	if err := uc.hasher.Compare(user.PasswordHash, in.OldPassword); err != nil {
		return fmt.Errorf("change_password: %w", model.ErrUnauthenticated)
	}

	newHash, err := uc.hasher.Hash(in.NewPassword)
	if err != nil {
		return fmt.Errorf("change_password: hash password: %w", err)
	}

	now := uc.clock.Now()
	err = uc.txRunner.WithinTx(ctx, func(ctx context.Context) error {
		user.PasswordHash = newHash
		user.UpdatedAt = now
		if err := uc.users.Update(ctx, user); err != nil {
			return fmt.Errorf("update user: %w", err)
		}
		if err := uc.sessions.RevokeAllExcept(ctx, user.ID, in.SessionID, now); err != nil {
			return fmt.Errorf("revoke sessions: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("change_password: %w", err)
	}

	if err := uc.events.Publish(ctx, "auth.password.changed", event.Marshal("auth.password.changed", "user", nil)); err != nil {
		slog.WarnContext(ctx, "change_password: publish event failed (best-effort)", "err", err)
	}
	return nil
}
