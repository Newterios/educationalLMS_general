package usecase

import (
	"context"
	"fmt"

	"github.com/Newterios/lms-system-prob/services/auth/internal/model"
	"github.com/Newterios/lms-system-prob/services/auth/internal/usecase/port"
	"github.com/google/uuid"
)

type ListSessionsUseCase struct {
	sessions port.SessionRepository
}

func NewListSessionsUseCase(sessions port.SessionRepository) *ListSessionsUseCase {
	return &ListSessionsUseCase{sessions: sessions}
}

type ListSessionsInput struct {
	UserID    uuid.UUID
	CurrentID uuid.UUID // ID of the session that issued the request (to mark current=true in transport layer)
}

type ListSessionsOutput struct {
	Sessions []*model.Session
}

func (uc *ListSessionsUseCase) Execute(ctx context.Context, in ListSessionsInput) (ListSessionsOutput, error) {
	sessions, err := uc.sessions.ListActiveForUser(ctx, in.UserID)
	if err != nil {
		return ListSessionsOutput{}, fmt.Errorf("list_sessions: %w", err)
	}
	return ListSessionsOutput{Sessions: sessions}, nil
}
