package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/Newterios/lms-system-prob/services/auth/internal/usecase"
	"github.com/Newterios/lms-system-prob/services/auth/internal/usecase/usecasetest"
	"github.com/google/uuid"
)

func TestListSessions_ReturnActiveOnly(t *testing.T) {
	sessions := usecasetest.NewFakeSessionRepository()
	uc := usecase.NewListSessionsUseCase(sessions)

	userID, _ := uuid.NewV7()
	now := time.Now()

	// two active sessions
	for i := 0; i < 2; i++ {
		id, _ := uuid.NewV7()
		seedSession(t, sessions, userID, "tok"+string(rune('a'+i)))
		_ = id
	}

	// one revoked session
	revokedSess := seedSession(t, sessions, userID, "revoked.tok")
	_ = sessions.Revoke(context.Background(), revokedSess.ID, now)

	out, err := uc.Execute(context.Background(), usecase.ListSessionsInput{
		UserID:    userID,
		CurrentID: uuid.New(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Sessions) != 2 {
		t.Errorf("expected 2 active sessions, got %d", len(out.Sessions))
	}
}

func TestListSessions_Empty(t *testing.T) {
	sessions := usecasetest.NewFakeSessionRepository()
	uc := usecase.NewListSessionsUseCase(sessions)

	out, err := uc.Execute(context.Background(), usecase.ListSessionsInput{UserID: uuid.New()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Sessions) != 0 {
		t.Errorf("expected empty list, got %d", len(out.Sessions))
	}
}
