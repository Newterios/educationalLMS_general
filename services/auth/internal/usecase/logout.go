package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Newterios/lms-system-prob/services/auth/internal/event"
	"github.com/Newterios/lms-system-prob/services/auth/internal/model"
	"github.com/Newterios/lms-system-prob/services/auth/internal/usecase/port"
)

type LogoutUseCase struct {
	sessions port.SessionRepository
	events   port.EventPublisher
	clock    port.Clock
}

func NewLogoutUseCase(sessions port.SessionRepository, events port.EventPublisher, clock port.Clock) *LogoutUseCase {
	return &LogoutUseCase{sessions: sessions, events: events, clock: clock}
}

type LogoutInput struct {
	RefreshToken string
}

func (uc *LogoutUseCase) Execute(ctx context.Context, in LogoutInput) error {
	session, err := uc.sessions.GetByRefreshHash(ctx, sha256Hex(in.RefreshToken))
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil // idempotent: already gone
		}
		return fmt.Errorf("logout: get session: %w", err)
	}

	if session.RevokedAt != nil {
		return nil // already revoked
	}

	if err := uc.sessions.Revoke(ctx, session.ID, uc.clock.Now()); err != nil {
		return fmt.Errorf("logout: revoke session: %w", err)
	}

	if err := uc.events.Publish(ctx, "auth.session.revoked",
		event.Marshal("auth.session.revoked", session.ID.String(), nil)); err != nil {
		slog.WarnContext(ctx, "logout: publish event failed (best-effort)", "err", err)
	}
	return nil
}
