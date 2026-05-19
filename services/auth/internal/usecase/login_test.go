package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Newterios/lms-system-prob/services/auth/internal/model"
	"github.com/Newterios/lms-system-prob/services/auth/internal/usecase"
	"github.com/Newterios/lms-system-prob/services/auth/internal/usecase/usecasetest"
	"github.com/google/uuid"
)

func seedUser(t *testing.T, users *usecasetest.FakeUserRepository, email, hashedPwd string) *model.User {
	t.Helper()
	id, _ := uuid.NewV7()
	u := &model.User{
		ID:           id,
		Email:        email,
		PasswordHash: hashedPwd,
		Role:         "student",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := users.Create(context.Background(), u); err != nil {
		t.Fatalf("seedUser: %v", err)
	}
	return u
}

func newLoginUC(t *testing.T) (*usecase.LoginUseCase, *usecasetest.FakeUserRepository, *usecasetest.FakeSessionRepository, *usecasetest.FakeTokenSigner) {
	t.Helper()
	users := usecasetest.NewFakeUserRepository()
	sessions := usecasetest.NewFakeSessionRepository()
	hasher := usecasetest.NewFakePasswordHasher()
	signer := &usecasetest.FakeTokenSigner{
		AccessTokenVal:  "access.token",
		RefreshTokenVal: "refresh.token",
		AccessExp:       time.Now().Add(15 * time.Minute),
		RefreshExp:      time.Now().Add(7 * 24 * time.Hour),
	}
	cache := usecasetest.NewFakeCache()
	clock := usecasetest.NewFakeClock(time.Now())
	uc := usecase.NewLoginUseCase(users, sessions, hasher, signer, cache, clock)
	return uc, users, sessions, signer
}

func TestLogin_HappyPath(t *testing.T) {
	uc, users, sessions, _ := newLoginUC(t)

	u := seedUser(t, users, "alice@example.com", "hashed:correctpass")

	out, err := uc.Execute(context.Background(), usecase.LoginInput{
		Email:    "alice@example.com",
		Password: "correctpass",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.AccessToken != "access.token" {
		t.Errorf("unexpected access token: %s", out.AccessToken)
	}
	if out.RefreshToken != "refresh.token" {
		t.Errorf("unexpected refresh token: %s", out.RefreshToken)
	}
	if out.AccessExpiresAt == "" {
		t.Error("expected non-empty AccessExpiresAt")
	}

	active, err := sessions.ListActiveForUser(context.Background(), u.ID)
	if err != nil || len(active) != 1 {
		t.Errorf("expected 1 active session, got %d (err=%v)", len(active), err)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	uc, users, _, _ := newLoginUC(t)
	seedUser(t, users, "bob@example.com", "hashed:correctpass")

	_, err := uc.Execute(context.Background(), usecase.LoginInput{
		Email:    "bob@example.com",
		Password: "wrongpass",
	})
	if !errors.Is(err, model.ErrUnauthenticated) {
		t.Errorf("expected ErrUnauthenticated, got %v", err)
	}
}

func TestLogin_UnknownEmail(t *testing.T) {
	uc, _, _, _ := newLoginUC(t)

	_, err := uc.Execute(context.Background(), usecase.LoginInput{
		Email:    "ghost@example.com",
		Password: "anypass12",
	})
	if !errors.Is(err, model.ErrUnauthenticated) {
		t.Errorf("expected ErrUnauthenticated (not ErrNotFound), got %v", err)
	}
	if errors.Is(err, model.ErrNotFound) {
		t.Error("must not expose ErrNotFound to the caller")
	}
}
