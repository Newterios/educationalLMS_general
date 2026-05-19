//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Newterios/lms-system-prob/services/auth/internal/model"
	"github.com/Newterios/lms-system-prob/services/auth/internal/repository/postgres"
	"github.com/google/uuid"
)

func TestVerificationCodeRepository_CreateAndGet(t *testing.T) {
	pool := newTestPool(t)
	userRepo := postgres.NewUserRepository(pool)
	codeRepo := postgres.NewVerificationCodeRepository(pool)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Millisecond)
	userID, _ := uuid.NewV7()
	_ = userRepo.Create(ctx, &model.User{
		ID: userID, Email: "vc@example.com", PasswordHash: "$2a$12$fake",
		Role: "student", CreatedAt: now, UpdatedAt: now,
	})

	codeID, _ := uuid.NewV7()
	code := &model.VerificationCode{
		ID:        codeID,
		UserID:    userID,
		Kind:      "email",
		CodeHash:  "sha256ofrawcode",
		ExpiresAt: now.Add(24 * time.Hour),
	}

	if err := codeRepo.Create(ctx, code); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := codeRepo.GetByCodeHash(ctx, "sha256ofrawcode")
	if err != nil {
		t.Fatalf("GetByCodeHash: %v", err)
	}
	if got.Kind != "email" {
		t.Errorf("Kind mismatch: %s", got.Kind)
	}
	if got.UsedAt != nil {
		t.Error("UsedAt should be nil initially")
	}
}

func TestVerificationCodeRepository_MarkUsed(t *testing.T) {
	pool := newTestPool(t)
	userRepo := postgres.NewUserRepository(pool)
	codeRepo := postgres.NewVerificationCodeRepository(pool)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Millisecond)
	userID, _ := uuid.NewV7()
	_ = userRepo.Create(ctx, &model.User{
		ID: userID, Email: "vc2@example.com", PasswordHash: "$2a$12$fake",
		Role: "student", CreatedAt: now, UpdatedAt: now,
	})

	codeID, _ := uuid.NewV7()
	_ = codeRepo.Create(ctx, &model.VerificationCode{
		ID:        codeID,
		UserID:    userID,
		Kind:      "password_reset",
		CodeHash:  "resethash",
		ExpiresAt: now.Add(time.Hour),
	})

	if err := codeRepo.MarkUsed(ctx, codeID, now); err != nil {
		t.Fatalf("MarkUsed: %v", err)
	}

	got, _ := codeRepo.GetByCodeHash(ctx, "resethash")
	if got.UsedAt == nil {
		t.Error("UsedAt should be set after MarkUsed")
	}
}

func TestVerificationCodeRepository_GetByCodeHash_NotFound(t *testing.T) {
	pool := newTestPool(t)
	repo := postgres.NewVerificationCodeRepository(pool)

	_, err := repo.GetByCodeHash(context.Background(), "nosuchhash")
	if !errors.Is(err, model.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
