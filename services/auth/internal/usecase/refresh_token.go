package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/Newterios/lms-system-prob/services/auth/internal/model"
	"github.com/Newterios/lms-system-prob/services/auth/internal/usecase/port"
	"github.com/google/uuid"
)

type RefreshTokenUseCase struct {
	users    port.UserRepository
	sessions port.SessionRepository
	signer   port.TokenSigner
	cache    port.Cache
	clock    port.Clock
}

func NewRefreshTokenUseCase(
	users port.UserRepository,
	sessions port.SessionRepository,
	signer port.TokenSigner,
	cache port.Cache,
	clock port.Clock,
) *RefreshTokenUseCase {
	return &RefreshTokenUseCase{users: users, sessions: sessions, signer: signer, cache: cache, clock: clock}
}

type RefreshTokenInput struct {
	RefreshToken string
	UserAgent    string
	IP           string
}

type RefreshTokenOutput struct {
	AccessToken     string
	RefreshToken    string
	AccessExpiresAt string // RFC3339
}

func (uc *RefreshTokenUseCase) Execute(ctx context.Context, in RefreshTokenInput) (RefreshTokenOutput, error) {
	claims, err := uc.signer.ParseRefresh(in.RefreshToken)
	if err != nil {
		return RefreshTokenOutput{}, fmt.Errorf("refresh: %w", model.ErrUnauthenticated)
	}

	oldSession, err := uc.sessions.GetByRefreshHash(ctx, sha256Hex(in.RefreshToken))
	if err != nil {
		return RefreshTokenOutput{}, fmt.Errorf("refresh: %w", model.ErrUnauthenticated)
	}

	now := uc.clock.Now()
	if oldSession.RevokedAt != nil {
		return RefreshTokenOutput{}, fmt.Errorf("refresh: session revoked: %w", model.ErrUnauthenticated)
	}
	if now.After(oldSession.ExpiresAt) {
		return RefreshTokenOutput{}, fmt.Errorf("refresh: session expired: %w", model.ErrUnauthenticated)
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return RefreshTokenOutput{}, fmt.Errorf("refresh: parse user id: %w", model.ErrUnauthenticated)
	}

	user, err := uc.users.GetByID(ctx, userID)
	if err != nil {
		return RefreshTokenOutput{}, fmt.Errorf("refresh: get user: %w", model.ErrUnauthenticated)
	}

	if err := uc.sessions.Revoke(ctx, oldSession.ID, now); err != nil {
		return RefreshTokenOutput{}, fmt.Errorf("refresh: revoke old session: %w", err)
	}

	newSessionID, err := uuid.NewV7()
	if err != nil {
		return RefreshTokenOutput{}, fmt.Errorf("refresh: new session id: %w", err)
	}

	accessToken, accessExp, err := uc.signer.SignAccess(user.ID.String(), newSessionID.String(), user.Role, user.EmailVerified)
	if err != nil {
		return RefreshTokenOutput{}, fmt.Errorf("refresh: sign access: %w", err)
	}

	refreshRaw, refreshExp, err := uc.signer.SignRefresh(user.ID.String(), newSessionID.String())
	if err != nil {
		return RefreshTokenOutput{}, fmt.Errorf("refresh: sign refresh: %w", err)
	}

	newSession := &model.Session{
		ID:          newSessionID,
		UserID:      user.ID,
		RefreshHash: sha256Hex(refreshRaw),
		UserAgent:   in.UserAgent,
		IP:          in.IP,
		ExpiresAt:   refreshExp,
		CreatedAt:   now,
	}
	if err := uc.sessions.Create(ctx, newSession); err != nil {
		return RefreshTokenOutput{}, fmt.Errorf("refresh: create session: %w", err)
	}

	return RefreshTokenOutput{
		AccessToken:     accessToken,
		RefreshToken:    refreshRaw,
		AccessExpiresAt: accessExp.Format(time.RFC3339),
	}, nil
}
