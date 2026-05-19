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

func TestConfirmPasswordReset_HappyPath(t *testing.T) {
	codes := usecasetest.NewFakeVerificationCodeRepository()
	users := usecasetest.NewFakeUserRepository()
	sessions := usecasetest.NewFakeSessionRepository()
	hasher := usecasetest.NewFakePasswordHasher()
	events := &usecasetest.FakeEventPublisher{}
	txRunner := &usecasetest.FakeTxRunner{}
	now := time.Now()
	clock := usecasetest.NewFakeClock(now)

	uc := usecase.NewConfirmPasswordResetUseCase(codes, users, sessions, hasher, events, txRunner, clock)

	userID, _ := uuid.NewV7()
	_ = users.Create(context.Background(), &model.User{
		ID:           userID,
		Email:        "u@x.com",
		PasswordHash: "hashed:oldpassword",
		Role:         "student",
		CreatedAt:    now,
		UpdatedAt:    now,
	})

	// plant an active session that should be revoked
	sessionID, _ := uuid.NewV7()
	_ = sessions.Create(context.Background(), &model.Session{
		ID:        sessionID,
		UserID:    userID,
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
	})

	codeID, _ := uuid.NewV7()
	_ = codes.Create(context.Background(), &model.VerificationCode{
		ID:        codeID,
		UserID:    userID,
		Kind:      "password_reset",
		CodeHash:  sha256Hex("resetcode"),
		ExpiresAt: now.Add(time.Hour),
	})

	err := uc.Execute(context.Background(), usecase.ConfirmPasswordResetInput{
		Code:        "resetcode",
		NewPassword: "newpassword",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// password updated
	u, _ := users.GetByID(context.Background(), userID)
	if u.PasswordHash != "hashed:newpassword" {
		t.Errorf("password not updated, got %s", u.PasswordHash)
	}

	// all sessions revoked
	active, _ := sessions.ListActiveForUser(context.Background(), userID)
	if len(active) != 0 {
		t.Errorf("expected all sessions revoked, got %d active", len(active))
	}

	// code marked used
	c, _ := codes.GetByCodeHash(context.Background(), sha256Hex("resetcode"))
	if c.UsedAt == nil {
		t.Error("code should be marked used")
	}
}

func TestConfirmPasswordReset_ShortPassword(t *testing.T) {
	uc := usecase.NewConfirmPasswordResetUseCase(
		usecasetest.NewFakeVerificationCodeRepository(),
		usecasetest.NewFakeUserRepository(),
		usecasetest.NewFakeSessionRepository(),
		usecasetest.NewFakePasswordHasher(),
		&usecasetest.FakeEventPublisher{},
		&usecasetest.FakeTxRunner{},
		usecasetest.NewFakeClock(time.Now()),
	)

	err := uc.Execute(context.Background(), usecase.ConfirmPasswordResetInput{
		Code:        "any",
		NewPassword: "short",
	})
	if !errors.Is(err, model.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestConfirmPasswordReset_AlreadyUsedCode(t *testing.T) {
	codes := usecasetest.NewFakeVerificationCodeRepository()
	now := time.Now()

	userID, _ := uuid.NewV7()
	codeID, _ := uuid.NewV7()
	used := now.Add(-time.Minute)
	_ = codes.Create(context.Background(), &model.VerificationCode{
		ID:        codeID,
		UserID:    userID,
		Kind:      "password_reset",
		CodeHash:  sha256Hex("usedcode"),
		ExpiresAt: now.Add(time.Hour),
		UsedAt:    &used,
	})

	uc := usecase.NewConfirmPasswordResetUseCase(
		codes,
		usecasetest.NewFakeUserRepository(),
		usecasetest.NewFakeSessionRepository(),
		usecasetest.NewFakePasswordHasher(),
		&usecasetest.FakeEventPublisher{},
		&usecasetest.FakeTxRunner{},
		usecasetest.NewFakeClock(now),
	)

	err := uc.Execute(context.Background(), usecase.ConfirmPasswordResetInput{
		Code:        "usedcode",
		NewPassword: "newpassword",
	})
	if !errors.Is(err, model.ErrFailedPrecondition) {
		t.Errorf("expected ErrFailedPrecondition, got %v", err)
	}
}
