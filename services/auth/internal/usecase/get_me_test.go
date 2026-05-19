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

func TestGetMe_HappyPath(t *testing.T) {
	users := usecasetest.NewFakeUserRepository()
	cache := usecasetest.NewFakeCache()
	uc := usecase.NewGetMeUseCase(users, cache)

	now := time.Now()
	userID, _ := uuid.NewV7()
	_ = users.Create(context.Background(), &model.User{
		ID:        userID,
		Email:     "me@x.com",
		FullName:  "Me",
		Role:      "student",
		CreatedAt: now,
		UpdatedAt: now,
	})

	out, err := uc.Execute(context.Background(), usecase.GetMeInput{UserID: userID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.User.ID != userID {
		t.Errorf("unexpected user ID: %s", out.User.ID)
	}
}

func TestGetMe_ServedFromCache(t *testing.T) {
	users := usecasetest.NewFakeUserRepository()
	cache := usecasetest.NewFakeCache()
	uc := usecase.NewGetMeUseCase(users, cache)

	now := time.Now()
	userID, _ := uuid.NewV7()
	_ = users.Create(context.Background(), &model.User{
		ID:        userID,
		Email:     "cache@x.com",
		Role:      "student",
		CreatedAt: now,
		UpdatedAt: now,
	})

	// first call populates cache
	if _, err := uc.Execute(context.Background(), usecase.GetMeInput{UserID: userID}); err != nil {
		t.Fatalf("first call: %v", err)
	}

	// delete from DB to prove second call is served from cache
	users.ForceGetErr = model.ErrNotFound

	out, err := uc.Execute(context.Background(), usecase.GetMeInput{UserID: userID})
	if err != nil {
		t.Fatalf("second call (should be cached): %v", err)
	}
	if out.User.ID != userID {
		t.Error("expected cached user to be returned")
	}
}

func TestGetMe_NotFound(t *testing.T) {
	users := usecasetest.NewFakeUserRepository()
	cache := usecasetest.NewFakeCache()
	uc := usecase.NewGetMeUseCase(users, cache)

	_, err := uc.Execute(context.Background(), usecase.GetMeInput{UserID: uuid.New()})
	if !errors.Is(err, model.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
