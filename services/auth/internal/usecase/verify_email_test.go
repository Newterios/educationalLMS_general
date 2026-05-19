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

func TestVerifyEmail_HappyPath(t *testing.T) {
	codes := usecasetest.NewFakeVerificationCodeRepository()
	users := usecasetest.NewFakeUserRepository()
	events := &usecasetest.FakeEventPublisher{}
	now := time.Now()
	clock := usecasetest.NewFakeClock(now)
	uc := usecase.NewVerifyEmailUseCase(codes, users, events, clock)

	userID, _ := uuid.NewV7()
	_ = users.Create(context.Background(), &model.User{ID: userID, Email: "u@x.com", Role: "student", CreatedAt: now, UpdatedAt: now})

	const rawCode = "myverifycode"
	codeID, _ := uuid.NewV7()
	_ = codes.Create(context.Background(), &model.VerificationCode{
		ID:        codeID,
		UserID:    userID,
		Kind:      "email",
		CodeHash:  sha256Hex(rawCode),
		ExpiresAt: now.Add(time.Hour),
	})

	err := uc.Execute(context.Background(), usecase.VerifyEmailInput{Code: rawCode})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	u, _ := users.GetByID(context.Background(), userID)
	if !u.EmailVerified {
		t.Error("expected EmailVerified=true after verify")
	}
}

func TestVerifyEmail_InvalidCode(t *testing.T) {
	codes := usecasetest.NewFakeVerificationCodeRepository()
	users := usecasetest.NewFakeUserRepository()
	uc := usecase.NewVerifyEmailUseCase(codes, users, &usecasetest.FakeEventPublisher{}, usecasetest.NewFakeClock(time.Now()))

	err := uc.Execute(context.Background(), usecase.VerifyEmailInput{Code: "nosuchcode"})
	if !errors.Is(err, model.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestVerifyEmail_ExpiredCode(t *testing.T) {
	codes := usecasetest.NewFakeVerificationCodeRepository()
	users := usecasetest.NewFakeUserRepository()
	now := time.Now()
	uc := usecase.NewVerifyEmailUseCase(codes, users, &usecasetest.FakeEventPublisher{}, usecasetest.NewFakeClock(now))

	userID, _ := uuid.NewV7()
	codeID, _ := uuid.NewV7()
	_ = codes.Create(context.Background(), &model.VerificationCode{
		ID:        codeID,
		UserID:    userID,
		Kind:      "email",
		CodeHash:  sha256Hex("expcode"),
		ExpiresAt: now.Add(-time.Hour), // already expired
	})

	err := uc.Execute(context.Background(), usecase.VerifyEmailInput{Code: "expcode"})
	if !errors.Is(err, model.ErrFailedPrecondition) {
		t.Errorf("expected ErrFailedPrecondition, got %v", err)
	}
}

func TestVerifyEmail_WrongKind(t *testing.T) {
	codes := usecasetest.NewFakeVerificationCodeRepository()
	users := usecasetest.NewFakeUserRepository()
	now := time.Now()
	uc := usecase.NewVerifyEmailUseCase(codes, users, &usecasetest.FakeEventPublisher{}, usecasetest.NewFakeClock(now))

	userID, _ := uuid.NewV7()
	codeID, _ := uuid.NewV7()
	_ = codes.Create(context.Background(), &model.VerificationCode{
		ID:        codeID,
		UserID:    userID,
		Kind:      "password_reset", // wrong kind
		CodeHash:  sha256Hex("wrongkind"),
		ExpiresAt: now.Add(time.Hour),
	})

	err := uc.Execute(context.Background(), usecase.VerifyEmailInput{Code: "wrongkind"})
	if !errors.Is(err, model.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}
