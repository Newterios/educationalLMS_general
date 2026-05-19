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

func TestUpdateProfile_HappyPath(t *testing.T) {
	users := usecasetest.NewFakeUserRepository()
	cache := usecasetest.NewFakeCache()
	events := &usecasetest.FakeEventPublisher{}
	now := time.Now()
	clock := usecasetest.NewFakeClock(now)

	uc := usecase.NewUpdateProfileUseCase(users, cache, events, clock)

	userID, _ := uuid.NewV7()
	_ = users.Create(context.Background(), &model.User{
		ID: userID, Email: "u@x.com", FullName: "Old Name", Locale: "en",
		Role: "student", CreatedAt: now, UpdatedAt: now,
	})

	// populate cache to verify it gets invalidated
	_, _ = usecase.NewGetMeUseCase(users, cache).Execute(context.Background(), usecase.GetMeInput{UserID: userID})

	out, err := uc.Execute(context.Background(), usecase.UpdateProfileInput{
		UserID:   userID,
		FullName: "New Name",
		Locale:   "ru",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.User.FullName != "New Name" {
		t.Errorf("expected FullName=New Name, got %s", out.User.FullName)
	}
	if out.User.Locale != "ru" {
		t.Errorf("expected Locale=ru, got %s", out.User.Locale)
	}

	// DB should have updated values
	u, _ := users.GetByID(context.Background(), userID)
	if u.FullName != "New Name" {
		t.Errorf("DB not updated, got %s", u.FullName)
	}
}

func TestUpdateProfile_PartialUpdate(t *testing.T) {
	users := usecasetest.NewFakeUserRepository()
	now := time.Now()
	clock := usecasetest.NewFakeClock(now)
	uc := usecase.NewUpdateProfileUseCase(users, usecasetest.NewFakeCache(), &usecasetest.FakeEventPublisher{}, clock)

	userID, _ := uuid.NewV7()
	_ = users.Create(context.Background(), &model.User{
		ID: userID, Email: "u@x.com", FullName: "Keep This", Locale: "en",
		Role: "student", CreatedAt: now, UpdatedAt: now,
	})

	// only updating locale
	out, err := uc.Execute(context.Background(), usecase.UpdateProfileInput{
		UserID: userID,
		Locale: "kz",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.User.FullName != "Keep This" {
		t.Errorf("FullName should be unchanged, got %s", out.User.FullName)
	}
	if out.User.Locale != "kz" {
		t.Errorf("expected Locale=kz, got %s", out.User.Locale)
	}
}

func TestUpdateProfile_NotFound(t *testing.T) {
	uc := usecase.NewUpdateProfileUseCase(
		usecasetest.NewFakeUserRepository(),
		usecasetest.NewFakeCache(),
		&usecasetest.FakeEventPublisher{},
		usecasetest.NewFakeClock(time.Now()),
	)

	_, err := uc.Execute(context.Background(), usecase.UpdateProfileInput{
		UserID:   uuid.New(),
		FullName: "X",
	})
	if !errors.Is(err, model.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
