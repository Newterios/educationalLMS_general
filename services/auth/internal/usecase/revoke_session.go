package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Newterios/lms-system-prob/services/auth/internal/event"
	"github.com/Newterios/lms-system-prob/services/auth/internal/model"
	"github.com/Newterios/lms-system-prob/services/auth/internal/usecase/port"
	"github.com/google/uuid"
)

type RevokeSessionUseCase struct {
	sessions port.SessionRepository
	events   port.EventPublisher
	clock    port.Clock
}

func NewRevokeSessionUseCase(sessions port.SessionRepository, events port.EventPublisher, clock port.Clock) *RevokeSessionUseCase {
	return &RevokeSessionUseCase{sessions: sessions, events: events, clock: clock}
}

type RevokeSessionInput struct {
	CallerID  uuid.UUID // from JWT — used to verify the session belongs to the caller
	SessionID uuid.UUID
}

func (uc *RevokeSessionUseCase) Execute(ctx context.Context, in RevokeSessionInput) error {
	session, err := uc.sessions.GetByID(ctx, in.SessionID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return fmt.Errorf("revoke_session: %w", model.ErrNotFound)
		}
		return fmt.Errorf("revoke_session: get session: %w", err)
	}

	if session.UserID != in.CallerID {
		return fmt.Errorf("revoke_session: %w", model.ErrPermissionDenied)
	}

	if session.RevokedAt != nil {
		return nil // idempotent
	}

	if err := uc.sessions.Revoke(ctx, session.ID, uc.clock.Now()); err != nil {
		return fmt.Errorf("revoke_session: %w", err)
	}

	if err := uc.events.Publish(ctx, "auth.session.revoked", event.Marshal("auth.session.revoked", session.ID.String(), nil)); err != nil {
		slog.WarnContext(ctx, "revoke_session: publish event failed (best-effort)", "err", err)
	}
	return nil
}
