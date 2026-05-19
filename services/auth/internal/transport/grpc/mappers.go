package grpc

import (
	"time"

	authv1 "github.com/Newterios/lms-system-prob/proto/auth/v1"
	"github.com/Newterios/lms-system-prob/services/auth/internal/model"
	"github.com/google/uuid"
)

// ── Domain → Proto ────────────────────────────────────────────────────────────

func userToProto(u *model.User) *authv1.User {
	return &authv1.User{
		Id:            u.ID.String(),
		Email:         u.Email,
		FullName:      u.FullName,
		Locale:        u.Locale,
		Role:          u.Role,
		EmailVerified: u.EmailVerified,
		CreatedAt:     u.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     u.UpdatedAt.Format(time.RFC3339),
	}
}

func sessionToProto(s *model.Session, currentID uuid.UUID) *authv1.Session {
	sess := &authv1.Session{
		Id:        s.ID.String(),
		UserAgent: s.UserAgent,
		Ip:        s.IP,
		CreatedAt: s.CreatedAt.Format(time.RFC3339),
		ExpiresAt: s.ExpiresAt.Format(time.RFC3339),
		Current:   s.ID == currentID,
	}
	return sess
}

// ── Proto → Domain (request mappers) ─────────────────────────────────────────

// parseUUID wraps uuid.Parse into a typed error suitable for toStatus.
func parseUUID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}, model.ErrInvalidInput
	}
	return id, nil
}

// extractPeer tries to get UserAgent and IP from gRPC metadata.
// gRPC clients send user-agent in "user-agent" metadata; peer addr carries IP.
func extractMeta(ua, ip string) (userAgent, remoteIP string) {
	return ua, ip
}
