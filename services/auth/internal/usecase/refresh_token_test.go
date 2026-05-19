package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Newterios/lms-system-prob/services/auth/internal/model"
	"github.com/Newterios/lms-system-prob/services/auth/internal/usecase"
	"github.com/Newterios/lms-system-prob/services/auth/internal/usecase/port"
	"github.com/Newterios/lms-system-prob/services/auth/internal/usecase/usecasetest"
	"github.com/google/uuid"
)

func newRefreshUC(t *testing.T) (*usecase.RefreshTokenUseCase, *usecasetest.FakeUserRepository, *usecasetest.FakeSessionRepository, *usecasetest.FakeTokenSigner) {
	t.Helper()
	users := usecasetest.NewFakeUserRepository()
	sessions := usecasetest.NewFakeSessionRepository()
	signer := &usecasetest.FakeTokenSigner{
		AccessTokenVal:  "new.access",
		RefreshTokenVal: "new.refresh",
		AccessExp:       time.Now().Add(15 * time.Minute),
		RefreshExp:      time.Now().Add(7 * 24 * time.Hour),
	}
	cache := usecasetest.NewFakeCache()
	clock := usecasetest.NewFakeClock(time.Now())
	uc := usecase.NewRefreshTokenUseCase(users, sessions, signer, cache, clock)
	return uc, users, sessions, signer
}

func TestRefreshToken_HappyPath(t *testing.T) {
	uc, users, sessions, signer := newRefreshUC(t)

	now := time.Now()
	userID, _ := uuid.NewV7()
	_ = users.Create(context.Background(), &model.User{
		ID: userID, Email: "r@x.com", Role: "student",
		CreatedAt: now, UpdatedAt: now,
	})

	const oldRefreshRaw = "old.refresh.token"
	signer.ParsedRefresh = port.RefreshClaims{UserID: userID.String(), SessionID: uuid.New().String()}

	oldSess := seedSession(t, sessions, userID, oldRefreshRaw)

	out, err := uc.Execute(context.Background(), usecase.RefreshTokenInput{
		RefreshToken: oldRefreshRaw,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.AccessToken != "new.access" {
		t.Errorf("unexpected access token: %s", out.AccessToken)
	}
	if out.RefreshToken != "new.refresh" {
		t.Errorf("unexpected refresh token: %s", out.RefreshToken)
	}

	// old session revoked
	old, _ := sessions.GetByID(context.Background(), oldSess.ID)
	if old.RevokedAt == nil {
		t.Error("old session should be revoked after rotation")
	}

	// new session created
	active, _ := sessions.ListActiveForUser(context.Background(), userID)
	if len(active) != 1 {
		t.Errorf("expected 1 active session after rotation, got %d", len(active))
	}
}

func TestRefreshToken_RevokedSession(t *testing.T) {
	uc, users, sessions, signer := newRefreshUC(t)

	now := time.Now()
	userID, _ := uuid.NewV7()
	_ = users.Create(context.Background(), &model.User{
		ID: userID, Email: "r2@x.com", Role: "student",
		CreatedAt: now, UpdatedAt: now,
	})

	const raw = "revoked.refresh"
	signer.ParsedRefresh = port.RefreshClaims{UserID: userID.String(), SessionID: uuid.New().String()}

	sess := seedSession(t, sessions, userID, raw)
	_ = sessions.Revoke(context.Background(), sess.ID, now)

	_, err := uc.Execute(context.Background(), usecase.RefreshTokenInput{RefreshToken: raw})
	if !errors.Is(err, model.ErrUnauthenticated) {
		t.Errorf("expected ErrUnauthenticated for revoked session, got %v", err)
	}
}

func TestRefreshToken_InvalidJWT(t *testing.T) {
	uc, _, _, signer := newRefreshUC(t)
	signer.ForceParseErr = model.ErrUnauthenticated

	_, err := uc.Execute(context.Background(), usecase.RefreshTokenInput{RefreshToken: "bad.jwt"})
	if !errors.Is(err, model.ErrUnauthenticated) {
		t.Errorf("expected ErrUnauthenticated, got %v", err)
	}
}
