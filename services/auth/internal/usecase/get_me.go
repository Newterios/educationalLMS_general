package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/Newterios/lms-system-prob/services/auth/internal/model"
	"github.com/Newterios/lms-system-prob/services/auth/internal/usecase/port"
	"github.com/google/uuid"
)

const userCacheTTL = 5 * time.Minute

type GetMeUseCase struct {
	users port.UserRepository
	cache port.Cache
}

func NewGetMeUseCase(users port.UserRepository, cache port.Cache) *GetMeUseCase {
	return &GetMeUseCase{users: users, cache: cache}
}

type GetMeInput struct {
	UserID uuid.UUID
}

type GetMeOutput struct {
	User *model.User
}

func (uc *GetMeUseCase) Execute(ctx context.Context, in GetMeInput) (GetMeOutput, error) {
	cacheKey := "user:" + in.UserID.String()

	if raw, err := uc.cache.Get(ctx, cacheKey); err == nil && raw != nil {
		var user model.User
		if err := json.Unmarshal(raw, &user); err == nil {
			return GetMeOutput{User: &user}, nil
		}
	}

	user, err := uc.users.GetByID(ctx, in.UserID)
	if err != nil {
		return GetMeOutput{}, fmt.Errorf("get_me: get user: %w", err)
	}

	if raw, err := json.Marshal(user); err == nil {
		if err := uc.cache.Set(ctx, cacheKey, raw, userCacheTTL); err != nil {
			slog.WarnContext(ctx, "get_me: cache set failed (best-effort)", "err", err)
		}
	}

	return GetMeOutput{User: user}, nil
}
