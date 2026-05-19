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

func TestChangePassword_HappyPath(t *testing.T) {
	users := usecasetest.NewFakeUserRepository()
	sessions := usecasetest.NewFakeSessionRepository()
	hasher := usecasetest.NewFakePasswordHasher()
	events := &usecasetest.FakeEventPublisher{}
	txRunner := &usecasetest.FakeTxRunner{}
	now := time.Now()
	clock := usecasetest.NewFakeClock(now)

	uc := usecase.NewChangePasswordUseCase(users, sessions, hasher, events, txRunner, clock)

	userID, _ := uuid.NewV7()
	_ = users.Create(context.Background(), &model.User{
		ID:           userID,
		Email:        "u@x.com",
		PasswordHash: "hashed:oldpass",
		Role:         "student",
		CreatedAt:    now,
		UpdatedAt:    now,
	})

	keepID, _ := uuid.NewV7()
	otherID, _ := uuid.NewV7()
	for _, id := range []uuid.UUID{keepID, otherID} {
		_ = sessions.Create(context.Background(), &model.Session{
			ID:        id,
			UserID:    userID,
			ExpiresAt: now.Add(time.Hour),
			CreatedAt: now,
		})
	}

	err := uc.Execute(context.Background(), usecase.ChangePasswordInput{
		UserID:      userID,
		SessionID:   keepID,
		OldPassword: "oldpass",
		NewPassword: "newpassword",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	u, _ := users.GetByID(context.Background(), userID)
	if u.PasswordHash != "hashed:newpassword" {
		t.Errorf("password not updated, got %s", u.PasswordHash)
	}

	// keepID session should remain active
	kept, _ := sessions.GetByID(context.Background(), keepID)
	if kept.RevokedAt != nil {
		t.Error("current session should not be revoked")
	}

	// otherID should be revoked
	other, _ := sessions.GetByID(context.Background(), otherID)
	if other.RevokedAt == nil {
		t.Error("other session should be revoked")
	}
}

func TestChangePassword_WrongOldPassword(t *testing.T) {
	users := usecasetest.NewFakeUserRepository()
	hasher := usecasetest.NewFakePasswordHasher()
	now := time.Now()
	userID, _ := uuid.NewV7()
	_ = users.Create(context.Background(), &model.User{
		ID:           userID,
		Email:        "u@x.com",
		PasswordHash: "hashed:correct",
		Role:         "student",
		CreatedAt:    now,
		UpdatedAt:    now,
	})

	uc := usecase.NewChangePasswordUseCase(
		users, usecasetest.NewFakeSessionRepository(), hasher,
		&usecasetest.FakeEventPublisher{}, &usecasetest.FakeTxRunner{},
		usecasetest.NewFakeClock(now),
	)

	err := uc.Execute(context.Background(), usecase.ChangePasswordInput{
		UserID:      userID,
		SessionID:   uuid.UUID{},
		OldPassword: "wrong",
		NewPassword: "newpassword",
	})
	if !errors.Is(err, model.ErrUnauthenticated) {
		t.Errorf("expected ErrUnauthenticated, got %v", err)
	}
}

func TestChangePassword_ShortNewPassword(t *testing.T) {
	uc := usecase.NewChangePasswordUseCase(
		usecasetest.NewFakeUserRepository(),
		usecasetest.NewFakeSessionRepository(),
		usecasetest.NewFakePasswordHasher(),
		&usecasetest.FakeEventPublisher{},
		&usecasetest.FakeTxRunner{},
		usecasetest.NewFakeClock(time.Now()),
	)

	_, _ = uuid.NewV7()
	err := uc.Execute(context.Background(), usecase.ChangePasswordInput{
		UserID:      uuid.New(),
		SessionID:   uuid.New(),
		OldPassword: "oldpass",
		NewPassword: "short",
	})
	if !errors.Is(err, model.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}
