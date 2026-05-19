package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Newterios/lms-system-prob/services/auth/internal/model"
	"github.com/Newterios/lms-system-prob/services/auth/internal/usecase/port"
	"github.com/google/uuid"
)

type LoginUseCase struct {
	users    port.UserRepository
	sessions port.SessionRepository
	hasher   port.PasswordHasher
	signer   port.TokenSigner
	cache    port.Cache
	clock    port.Clock
}

func NewLoginUseCase(
	users port.UserRepository,
	sessions port.SessionRepository,
	hasher port.PasswordHasher,
	signer port.TokenSigner,
	cache port.Cache,
	clock port.Clock,
) *LoginUseCase {
	return &LoginUseCase{users: users, sessions: sessions, hasher: hasher, signer: signer, cache: cache, clock: clock}
}

type LoginInput struct {
	Email     string
	Password  string
	UserAgent string
	IP        string
}

type LoginOutput struct {
	AccessToken     string
	RefreshToken    string
	AccessExpiresAt string // RFC3339
}

func (uc *LoginUseCase) Execute(ctx context.Context, in LoginInput) (LoginOutput, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))

	user, err := uc.users.GetByEmail(ctx, email)
	if err != nil {
		// never reveal whether the email exists
		return LoginOutput{}, fmt.Errorf("login: %w", model.ErrUnauthenticated)
	}

	if err := uc.hasher.Compare(user.PasswordHash, in.Password); err != nil {
		return LoginOutput{}, fmt.Errorf("login: %w", model.ErrUnauthenticated)
	}

	sessionID, err := uuid.NewV7()
	if err != nil {
		return LoginOutput{}, fmt.Errorf("login: new session id: %w", err)
	}

	accessToken, accessExp, err := uc.signer.SignAccess(user.ID.String(), sessionID.String(), user.Role, user.EmailVerified)
	if err != nil {
		return LoginOutput{}, fmt.Errorf("login: sign access: %w", err)
	}

	refreshRaw, refreshExp, err := uc.signer.SignRefresh(user.ID.String(), sessionID.String())
	if err != nil {
		return LoginOutput{}, fmt.Errorf("login: sign refresh: %w", err)
	}

	now := uc.clock.Now()
	session := &model.Session{
		ID:          sessionID,
		UserID:      user.ID,
		RefreshHash: sha256Hex(refreshRaw),
		UserAgent:   in.UserAgent,
		IP:          in.IP,
		ExpiresAt:   refreshExp,
		CreatedAt:   now,
	}
	if err := uc.sessions.Create(ctx, session); err != nil {
		return LoginOutput{}, fmt.Errorf("login: create session: %w", err)
	}

	if err := uc.cache.Delete(ctx, "user:"+user.ID.String()); err != nil {
		slog.WarnContext(ctx, "login: cache invalidation failed (best-effort)", "err", err)
	}

	return LoginOutput{
		AccessToken:     accessToken,
		RefreshToken:    refreshRaw,
		AccessExpiresAt: accessExp.Format(time.RFC3339),
	}, nil
}
