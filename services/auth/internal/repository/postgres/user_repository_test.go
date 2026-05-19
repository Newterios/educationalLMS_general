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

func TestUserRepository_CreateAndGet(t *testing.T) {
	pool := newTestPool(t)
	repo := postgres.NewUserRepository(pool)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Millisecond)
	id, _ := uuid.NewV7()
	user := &model.User{
		ID:            id,
		Email:         "alice@example.com",
		PasswordHash:  "$2a$12$fakehash",
		FullName:      "Alice",
		Locale:        "en",
		Role:          "student",
		EmailVerified: false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Email != user.Email {
		t.Errorf("email mismatch: got %s, want %s", got.Email, user.Email)
	}

	got2, err := repo.GetByEmail(ctx, "alice@example.com")
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}
	if got2.ID != id {
		t.Errorf("ID mismatch from GetByEmail")
	}
}

func TestUserRepository_Create_DuplicateEmail(t *testing.T) {
	pool := newTestPool(t)
	repo := postgres.NewUserRepository(pool)
	ctx := context.Background()

	now := time.Now().UTC()
	makeUser := func() *model.User {
		id, _ := uuid.NewV7()
		return &model.User{
			ID:           id,
			Email:        "dup@example.com",
			PasswordHash: "$2a$12$fake",
			Role:         "student",
			CreatedAt:    now,
			UpdatedAt:    now,
		}
	}

	if err := repo.Create(ctx, makeUser()); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if err := repo.Create(ctx, makeUser()); !errors.Is(err, model.ErrAlreadyExists) {
		t.Errorf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestUserRepository_GetByID_NotFound(t *testing.T) {
	pool := newTestPool(t)
	repo := postgres.NewUserRepository(pool)

	_, err := repo.GetByID(context.Background(), uuid.New())
	if !errors.Is(err, model.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUserRepository_Update(t *testing.T) {
	pool := newTestPool(t)
	repo := postgres.NewUserRepository(pool)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Millisecond)
	id, _ := uuid.NewV7()
	_ = repo.Create(ctx, &model.User{
		ID:           id,
		Email:        "upd@example.com",
		PasswordHash: "$2a$12$fake",
		Role:         "student",
		CreatedAt:    now,
		UpdatedAt:    now,
	})

	u, _ := repo.GetByID(ctx, id)
	u.FullName = "Updated Name"
	u.EmailVerified = true
	if err := repo.Update(ctx, u); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := repo.GetByID(ctx, id)
	if got.FullName != "Updated Name" {
		t.Errorf("FullName not updated: %s", got.FullName)
	}
	if !got.EmailVerified {
		t.Error("EmailVerified not updated")
	}
}
