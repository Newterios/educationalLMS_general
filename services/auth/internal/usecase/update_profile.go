package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Newterios/lms-system-prob/services/auth/internal/event"
	"github.com/Newterios/lms-system-prob/services/auth/internal/model"
	"github.com/Newterios/lms-system-prob/services/auth/internal/usecase/port"
	"github.com/google/uuid"
)

type UpdateProfileUseCase struct {
	users  port.UserRepository
	cache  port.Cache
	events port.EventPublisher
	clock  port.Clock
}

func NewUpdateProfileUseCase(
	users port.UserRepository,
	cache port.Cache,
	events port.EventPublisher,
	clock port.Clock,
) *UpdateProfileUseCase {
	return &UpdateProfileUseCase{users: users, cache: cache, events: events, clock: clock}
}

type UpdateProfileInput struct {
	UserID   uuid.UUID
	FullName string
	Locale   string
}

type UpdateProfileOutput struct {
	User *model.User
}

func (uc *UpdateProfileUseCase) Execute(ctx context.Context, in UpdateProfileInput) (UpdateProfileOutput, error) {
	user, err := uc.users.GetByID(ctx, in.UserID)
	if err != nil {
		return UpdateProfileOutput{}, fmt.Errorf("update_profile: get user: %w", err)
	}

	if in.FullName != "" {
		user.FullName = in.FullName
	}
	if in.Locale != "" {
		user.Locale = in.Locale
	}
	user.UpdatedAt = uc.clock.Now()

	if err := uc.users.Update(ctx, user); err != nil {
		return UpdateProfileOutput{}, fmt.Errorf("update_profile: update user: %w", err)
	}

	if err := uc.cache.Delete(ctx, "user:"+in.UserID.String()); err != nil {
		slog.WarnContext(ctx, "update_profile: cache delete failed (best-effort)", "err", err)
	}

	if err := uc.events.Publish(ctx, "auth.user.updated",
		event.Marshal("auth.user.updated", in.UserID.String(), map[string]string{
			"full_name": user.FullName,
		})); err != nil {
		slog.WarnContext(ctx, "update_profile: publish event failed (best-effort)", "err", err)
	}

	return UpdateProfileOutput{User: user}, nil
}
