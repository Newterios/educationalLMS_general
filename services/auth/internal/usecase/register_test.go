package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Newterios/lms-system-prob/services/auth/internal/model"
	"github.com/Newterios/lms-system-prob/services/auth/internal/usecase"
	"github.com/Newterios/lms-system-prob/services/auth/internal/usecase/usecasetest"
)

func newRegisterUC(t *testing.T) (*usecase.RegisterUseCase, *usecasetest.FakeUserRepository, *usecasetest.FakeVerificationCodeRepository, *usecasetest.FakeMailer) {
	t.Helper()
	users := usecasetest.NewFakeUserRepository()
	codes := usecasetest.NewFakeVerificationCodeRepository()
	codeGen := usecasetest.NewFakeCodeGenerator("rawcode", "hashcode")
	hasher := usecasetest.NewFakePasswordHasher()
	events := &usecasetest.FakeEventPublisher{}
	mailer := &usecasetest.FakeMailer{}
	clock := usecasetest.NewFakeClock(time.Now())
	uc := usecase.NewRegisterUseCase(users, codes, codeGen, hasher, events, mailer, clock)
	return uc, users, codes, mailer
}

func TestRegister_HappyPath(t *testing.T) {
	uc, users, codes, mailer := newRegisterUC(t)
	ctx := context.Background()

	out, err := uc.Execute(ctx, usecase.RegisterInput{
		Email:    "alice@example.com",
		Password: "secret12",
		FullName: "Alice",
		Locale:   "en",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.UserID == "" {
		t.Error("expected non-empty UserID")
	}
	if !out.RequiresEmailVerification {
		t.Error("expected RequiresEmailVerification=true")
	}

	u, err := users.GetByEmail(ctx, "alice@example.com")
	if err != nil {
		t.Fatalf("user not found: %v", err)
	}
	if u.EmailVerified {
		t.Error("user should not be verified yet")
	}
	if u.Role != "student" {
		t.Errorf("expected role=student, got %s", u.Role)
	}

	code, err := codes.GetByCodeHash(ctx, "hashcode")
	if err != nil {
		t.Fatalf("code not found: %v", err)
	}
	if code.Kind != "email" {
		t.Errorf("expected kind=email, got %s", code.Kind)
	}

	if len(mailer.VerificationEmails) != 1 {
		t.Errorf("expected 1 verification email, got %d", len(mailer.VerificationEmails))
	}
}

func TestRegister_InvalidEmail(t *testing.T) {
	uc, _, _, _ := newRegisterUC(t)

	_, err := uc.Execute(context.Background(), usecase.RegisterInput{
		Email:    "notanemail",
		Password: "secret12",
	})
	if !errors.Is(err, model.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestRegister_ShortPassword(t *testing.T) {
	uc, _, _, _ := newRegisterUC(t)

	_, err := uc.Execute(context.Background(), usecase.RegisterInput{
		Email:    "bob@example.com",
		Password: "short",
	})
	if !errors.Is(err, model.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	uc, users, _, _ := newRegisterUC(t)

	in := usecase.RegisterInput{Email: "dup@example.com", Password: "secret12", FullName: "Dup"}
	if _, err := uc.Execute(context.Background(), in); err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	users.ForceCreateErr = model.ErrAlreadyExists
	_, err := uc.Execute(context.Background(), in)
	if !errors.Is(err, model.ErrAlreadyExists) {
		t.Errorf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestRegister_EmailNormalized(t *testing.T) {
	uc, users, _, _ := newRegisterUC(t)

	_, err := uc.Execute(context.Background(), usecase.RegisterInput{
		Email:    "  Alice@Example.COM  ",
		Password: "secret12",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	u, err := users.GetByEmail(context.Background(), "alice@example.com")
	if err != nil {
		t.Fatalf("normalized email not found: %v", err)
	}
	if u.Email != "alice@example.com" {
		t.Errorf("expected lowercase email, got %s", u.Email)
	}
}
