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

func TestRevokeSession_HappyPath(t *testing.T) {
	sessions := usecasetest.NewFakeSessionRepository()
	clock := usecasetest.NewFakeClock(time.Now())
	uc := usecase.NewRevokeSessionUseCase(sessions, &usecasetest.FakeEventPublisher{}, clock)

	callerID, _ := uuid.NewV7()
	sessionID, _ := uuid.NewV7()
	_ = sessions.Create(context.Background(), &model.Session{
		ID:        sessionID,
		UserID:    callerID,
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	})

	err := uc.Execute(context.Background(), usecase.RevokeSessionInput{
		CallerID:  callerID,
		SessionID: sessionID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s, _ := sessions.GetByID(context.Background(), sessionID)
	if s.RevokedAt == nil {
		t.Error("session should be revoked")
	}
}

func TestRevokeSession_PermissionDenied(t *testing.T) {
	sessions := usecasetest.NewFakeSessionRepository()
	uc := usecase.NewRevokeSessionUseCase(sessions, &usecasetest.FakeEventPublisher{}, usecasetest.NewFakeClock(time.Now()))

	ownerID, _ := uuid.NewV7()
	attackerID, _ := uuid.NewV7()
	sessionID, _ := uuid.NewV7()
	_ = sessions.Create(context.Background(), &model.Session{
		ID:        sessionID,
		UserID:    ownerID,
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	})

	err := uc.Execute(context.Background(), usecase.RevokeSessionInput{
		CallerID:  attackerID,
		SessionID: sessionID,
	})
	if !errors.Is(err, model.ErrPermissionDenied) {
		t.Errorf("expected ErrPermissionDenied, got %v", err)
	}
}

func TestRevokeSession_NotFound(t *testing.T) {
	sessions := usecasetest.NewFakeSessionRepository()
	uc := usecase.NewRevokeSessionUseCase(sessions, &usecasetest.FakeEventPublisher{}, usecasetest.NewFakeClock(time.Now()))

	err := uc.Execute(context.Background(), usecase.RevokeSessionInput{
		CallerID:  uuid.New(),
		SessionID: uuid.New(),
	})
	if !errors.Is(err, model.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestRevokeSession_Idempotent(t *testing.T) {
	sessions := usecasetest.NewFakeSessionRepository()
	clock := usecasetest.NewFakeClock(time.Now())
	uc := usecase.NewRevokeSessionUseCase(sessions, &usecasetest.FakeEventPublisher{}, clock)

	callerID, _ := uuid.NewV7()
	sessionID, _ := uuid.NewV7()
	_ = sessions.Create(context.Background(), &model.Session{
		ID:        sessionID,
		UserID:    callerID,
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	})

	in := usecase.RevokeSessionInput{CallerID: callerID, SessionID: sessionID}
	if err := uc.Execute(context.Background(), in); err != nil {
		t.Fatalf("first revoke: %v", err)
	}
	if err := uc.Execute(context.Background(), in); err != nil {
		t.Fatalf("second revoke (idempotent): %v", err)
	}
}
