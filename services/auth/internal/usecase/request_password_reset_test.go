package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/Newterios/lms-system-prob/services/auth/internal/model"
	"github.com/Newterios/lms-system-prob/services/auth/internal/usecase"
	"github.com/Newterios/lms-system-prob/services/auth/internal/usecase/usecasetest"
	"github.com/google/uuid"
)

func TestRequestPasswordReset_HappyPath(t *testing.T) {
	users := usecasetest.NewFakeUserRepository()
	codes := usecasetest.NewFakeVerificationCodeRepository()
	events := &usecasetest.FakeEventPublisher{}
	mailer := &usecasetest.FakeMailer{}
	codeGen := usecasetest.NewFakeCodeGenerator("rawreset", "hashreset")
	now := time.Now()
	clock := usecasetest.NewFakeClock(now)

	uc := usecase.NewRequestPasswordResetUseCase(users, codes, events, mailer, codeGen, clock)

	userID, _ := uuid.NewV7()
	_ = users.Create(context.Background(), &model.User{
		ID: userID, Email: "pw@x.com", FullName: "PW User", Role: "student",
		CreatedAt: now, UpdatedAt: now,
	})

	err := uc.Execute(context.Background(), usecase.RequestPasswordResetInput{Email: "pw@x.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	code, err := codes.GetByCodeHash(context.Background(), "hashreset")
	if err != nil {
		t.Fatalf("code not created: %v", err)
	}
	if code.Kind != "password_reset" {
		t.Errorf("expected kind=password_reset, got %s", code.Kind)
	}

	if len(mailer.PasswordResetEmails) != 1 {
		t.Errorf("expected 1 password reset email, got %d", len(mailer.PasswordResetEmails))
	}
	if mailer.PasswordResetEmails[0].Code != "rawreset" {
		t.Errorf("expected raw code in email, got %s", mailer.PasswordResetEmails[0].Code)
	}
}

func TestRequestPasswordReset_UnknownEmail_NoError(t *testing.T) {
	// anti-enumeration: must return nil for unknown email
	uc := usecase.NewRequestPasswordResetUseCase(
		usecasetest.NewFakeUserRepository(),
		usecasetest.NewFakeVerificationCodeRepository(),
		&usecasetest.FakeEventPublisher{},
		&usecasetest.FakeMailer{},
		usecasetest.NewFakeCodeGenerator("r", "h"),
		usecasetest.NewFakeClock(time.Now()),
	)

	err := uc.Execute(context.Background(), usecase.RequestPasswordResetInput{Email: "nobody@x.com"})
	if err != nil {
		t.Errorf("expected nil for unknown email (anti-enumeration), got %v", err)
	}
}
