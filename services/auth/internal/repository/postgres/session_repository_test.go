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

func TestSessionRepository_CreateAndGet(t *testing.T) {
	pool := newTestPool(t)
	userRepo := postgres.NewUserRepository(pool)
	sessRepo := postgres.NewSessionRepository(pool)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Millisecond)
	userID, _ := uuid.NewV7()
	_ = userRepo.Create(ctx, &model.User{
		ID:           userID,
		Email:        "sess@example.com",
		PasswordHash: "$2a$12$fake",
		Role:         "student",
		CreatedAt:    now,
		UpdatedAt:    now,
	})

	sessID, _ := uuid.NewV7()
	sess := &model.Session{
		ID:          sessID,
		UserID:      userID,
		RefreshHash: "deadbeef1234",
		UserAgent:   "Go test",
		IP:          "127.0.0.1",
		ExpiresAt:   now.Add(7 * 24 * time.Hour),
		CreatedAt:   now,
	}

	if err := sessRepo.Create(ctx, sess); err != nil {
		t.Fatalf("Create session: %v", err)
	}

	got, err := sessRepo.GetByID(ctx, sessID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.UserID != userID {
		t.Errorf("UserID mismatch")
	}

	got2, err := sessRepo.GetByRefreshHash(ctx, "deadbeef1234")
	if err != nil {
		t.Fatalf("GetByRefreshHash: %v", err)
	}
	if got2.ID != sessID {
		t.Errorf("ID mismatch from GetByRefreshHash")
	}
}

func TestSessionRepository_Revoke(t *testing.T) {
	pool := newTestPool(t)
	userRepo := postgres.NewUserRepository(pool)
	sessRepo := postgres.NewSessionRepository(pool)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Millisecond)
	userID, _ := uuid.NewV7()
	_ = userRepo.Create(ctx, &model.User{
		ID: userID, Email: "rev@example.com", PasswordHash: "$2a$12$fake",
		Role: "student", CreatedAt: now, UpdatedAt: now,
	})

	sessID, _ := uuid.NewV7()
	_ = sessRepo.Create(ctx, &model.Session{
		ID:          sessID,
		UserID:      userID,
		RefreshHash: "revokehash",
		ExpiresAt:   now.Add(time.Hour),
		CreatedAt:   now,
	})

	if err := sessRepo.Revoke(ctx, sessID, now); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	// revoked sessions must not appear in ListActiveForUser
	active, err := sessRepo.ListActiveForUser(ctx, userID)
	if err != nil {
		t.Fatalf("ListActiveForUser: %v", err)
	}
	if len(active) != 0 {
		t.Errorf("expected 0 active sessions after revoke, got %d", len(active))
	}
}

func TestSessionRepository_RevokeAllForUser(t *testing.T) {
	pool := newTestPool(t)
	userRepo := postgres.NewUserRepository(pool)
	sessRepo := postgres.NewSessionRepository(pool)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Millisecond)
	userID, _ := uuid.NewV7()
	_ = userRepo.Create(ctx, &model.User{
		ID: userID, Email: "revokeall@example.com", PasswordHash: "$2a$12$fake",
		Role: "student", CreatedAt: now, UpdatedAt: now,
	})

	for i, h := range []string{"h1", "h2", "h3"} {
		id, _ := uuid.NewV7()
		_ = sessRepo.Create(ctx, &model.Session{
			ID:          id,
			UserID:      userID,
			RefreshHash: h,
			ExpiresAt:   now.Add(time.Hour),
			CreatedAt:   now.Add(time.Duration(i) * time.Second),
		})
	}

	if err := sessRepo.RevokeAllForUser(ctx, userID, now); err != nil {
		t.Fatalf("RevokeAllForUser: %v", err)
	}

	active, _ := sessRepo.ListActiveForUser(ctx, userID)
	if len(active) != 0 {
		t.Errorf("expected 0 active sessions, got %d", len(active))
	}
}

func TestSessionRepository_RevokeAllExcept(t *testing.T) {
	pool := newTestPool(t)
	userRepo := postgres.NewUserRepository(pool)
	sessRepo := postgres.NewSessionRepository(pool)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Millisecond)
	userID, _ := uuid.NewV7()
	_ = userRepo.Create(ctx, &model.User{
		ID: userID, Email: "except@example.com", PasswordHash: "$2a$12$fake",
		Role: "student", CreatedAt: now, UpdatedAt: now,
	})

	keepID, _ := uuid.NewV7()
	_ = sessRepo.Create(ctx, &model.Session{
		ID: keepID, UserID: userID, RefreshHash: "keep",
		ExpiresAt: now.Add(time.Hour), CreatedAt: now,
	})

	for _, h := range []string{"other1", "other2"} {
		id, _ := uuid.NewV7()
		_ = sessRepo.Create(ctx, &model.Session{
			ID: id, UserID: userID, RefreshHash: h,
			ExpiresAt: now.Add(time.Hour), CreatedAt: now,
		})
	}

	if err := sessRepo.RevokeAllExcept(ctx, userID, keepID, now); err != nil {
		t.Fatalf("RevokeAllExcept: %v", err)
	}

	active, _ := sessRepo.ListActiveForUser(ctx, userID)
	if len(active) != 1 || active[0].ID != keepID {
		t.Errorf("expected only keepID to remain active, got %d sessions", len(active))
	}
}

func TestSessionRepository_GetByID_NotFound(t *testing.T) {
	pool := newTestPool(t)
	repo := postgres.NewSessionRepository(pool)

	_, err := repo.GetByID(context.Background(), uuid.New())
	if !errors.Is(err, model.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
