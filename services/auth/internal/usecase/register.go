package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Newterios/lms-system-prob/services/auth/internal/event"
	"github.com/Newterios/lms-system-prob/services/auth/internal/model"
	"github.com/Newterios/lms-system-prob/services/auth/internal/usecase/port"
	"github.com/google/uuid"
)

type RegisterUseCase struct {
	users    port.UserRepository
	codeRepo port.VerificationCodeRepository
	codeGen  port.CodeGenerator
	hasher   port.PasswordHasher
	events   port.EventPublisher
	mailer   port.Mailer
	clock    port.Clock
}

func NewRegisterUseCase(
	users port.UserRepository,
	codeRepo port.VerificationCodeRepository,
	codeGen port.CodeGenerator,
	hasher port.PasswordHasher,
	events port.EventPublisher,
	mailer port.Mailer,
	clock port.Clock,
) *RegisterUseCase {
	return &RegisterUseCase{users: users, codeRepo: codeRepo, codeGen: codeGen,
		hasher: hasher, events: events, mailer: mailer, clock: clock}
}

type RegisterInput struct {
	Email    string
	Password string
	FullName string
	Locale   string
}

type RegisterOutput struct {
	UserID                    string
	RequiresEmailVerification bool
}

func (uc *RegisterUseCase) Execute(ctx context.Context, in RegisterInput) (RegisterOutput, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if !strings.Contains(email, "@") {
		return RegisterOutput{}, fmt.Errorf("register: %w: invalid email", model.ErrInvalidInput)
	}
	if len(in.Password) < 8 {
		return RegisterOutput{}, fmt.Errorf("register: %w: password must be at least 8 characters", model.ErrInvalidInput)
	}

	hash, err := uc.hasher.Hash(in.Password)
	if err != nil {
		return RegisterOutput{}, fmt.Errorf("register: hash password: %w", err)
	}

	now := uc.clock.Now()
	userID, err := uuid.NewV7()
	if err != nil {
		return RegisterOutput{}, fmt.Errorf("register: new user id: %w", err)
	}
	user := &model.User{
		ID:            userID,
		Email:         email,
		PasswordHash:  hash,
		FullName:      in.FullName,
		Locale:        in.Locale,
		Role:          "student",
		EmailVerified: false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := uc.users.Create(ctx, user); err != nil {
		return RegisterOutput{}, fmt.Errorf("register: create user: %w", err)
	}

	raw, codeHash, err := uc.codeGen.Generate()
	if err != nil {
		return RegisterOutput{}, fmt.Errorf("register: generate code: %w", err)
	}
	codeID, err := uuid.NewV7()
	if err != nil {
		return RegisterOutput{}, fmt.Errorf("register: new code id: %w", err)
	}
	code := &model.VerificationCode{
		ID:        codeID,
		UserID:    userID,
		Kind:      "email",
		CodeHash:  codeHash,
		ExpiresAt: now.Add(24 * time.Hour),
	}
	if err := uc.codeRepo.Create(ctx, code); err != nil {
		return RegisterOutput{}, fmt.Errorf("register: create verification code: %w", err)
	}

	payload := event.Marshal("auth.user.registered", userID.String(), map[string]any{
		"email":     email,
		"full_name": in.FullName,
		"locale":    in.Locale,
	})
	if err := uc.events.Publish(ctx, "auth.user.registered", payload); err != nil {
		slog.WarnContext(ctx, "register: publish event failed (best-effort)", "err", err)
	}
	if err := uc.mailer.SendVerificationEmail(ctx, email, in.FullName, raw); err != nil {
		slog.WarnContext(ctx, "register: send verification email failed (best-effort)", "err", err)
	}

	return RegisterOutput{UserID: userID.String(), RequiresEmailVerification: true}, nil
}
