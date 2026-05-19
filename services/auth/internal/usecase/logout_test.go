package usecase_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/Newterios/lms-system-prob/services/auth/internal/model"
	"github.com/Newterios/lms-system-prob/services/auth/internal/usecase"
	"github.com/Newterios/lms-system-prob/services/auth/internal/usecase/usecasetest"
	"github.com/google/uuid"
)

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func seedSession(t *testing.T, sessions *usecasetest.FakeSessionRepository, userID uuid.UUID, refreshRaw string) *model.Session {
	t.Helper()
	id, _ := uuid.NewV7()
	s := &model.Session{
		ID:          id,
		UserID:      userID,
		RefreshHash: sha256Hex(refreshRaw),
		ExpiresAt:   time.Now().Add(7 * 24 * time.Hour),
		CreatedAt:   time.Now(),
	}
	if err := sessions.Create(context.Background(), s); err != nil {
		t.Fatalf("seedSession: %v", err)
	}
	return s
}

func TestLogout_HappyPath(t *testing.T) {
	sessions := usecasetest.NewFakeSessionRepository()
	events := &usecasetest.FakeEventPublisher{}
	clock := usecasetest.NewFakeClock(time.Now())
	uc := usecase.NewLogoutUseCase(sessions, events, clock)

	userID, _ := uuid.NewV7()
	seedSession(t, sessions, userID, "my.refresh.token")

	err := uc.Execute(context.Background(), usecase.LogoutInput{RefreshToken: "my.refresh.token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// session should be revoked
	s, _ := sessions.GetByRefreshHash(context.Background(), sha256Hex("my.refresh.token"))
	if s.RevokedAt == nil {
		t.Error("session should be revoked")
	}
}

func TestLogout_UnknownToken_Idempotent(t *testing.T) {
	sessions := usecasetest.NewFakeSessionRepository()
	uc := usecase.NewLogoutUseCase(sessions, &usecasetest.FakeEventPublisher{}, usecasetest.NewFakeClock(time.Now()))

	err := uc.Execute(context.Background(), usecase.LogoutInput{RefreshToken: "unknown.token"})
	if err != nil {
		t.Errorf("expected nil for unknown token, got %v", err)
	}
}

func TestLogout_AlreadyRevoked_Idempotent(t *testing.T) {
	sessions := usecasetest.NewFakeSessionRepository()
	clock := usecasetest.NewFakeClock(time.Now())
	uc := usecase.NewLogoutUseCase(sessions, &usecasetest.FakeEventPublisher{}, clock)

	userID, _ := uuid.NewV7()
	s := seedSession(t, sessions, userID, "tok")
	_ = sessions.Revoke(context.Background(), s.ID, time.Now())

	err := uc.Execute(context.Background(), usecase.LogoutInput{RefreshToken: "tok"})
	if err != nil {
		t.Errorf("expected nil for already-revoked session, got %v", err)
	}
}
