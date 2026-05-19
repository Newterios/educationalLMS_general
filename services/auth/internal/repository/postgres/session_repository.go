package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Newterios/lms-system-prob/services/auth/internal/model"
	"github.com/google/uuid"
)

type sessionRepository struct{ pool *pgxpool.Pool }

func NewSessionRepository(pool *pgxpool.Pool) *sessionRepository {
	return &sessionRepository{pool: pool}
}

func (r *sessionRepository) Create(ctx context.Context, s *model.Session) error {
	_, err := db(ctx, r.pool).Exec(ctx, `
		INSERT INTO sessions
			(id, user_id, refresh_hash, user_agent, ip, expires_at, created_at)
		VALUES ($1,$2,$3,$4,$5::inet,$6,$7)`,
		s.ID, s.UserID, s.RefreshHash, s.UserAgent, s.IP, s.ExpiresAt, s.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (r *sessionRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Session, error) {
	row := db(ctx, r.pool).QueryRow(ctx, `
		SELECT id, user_id, refresh_hash, user_agent, ip::text,
		       expires_at, created_at, revoked_at
		FROM sessions WHERE id = $1`, id)
	return scanSession(row)
}

func (r *sessionRepository) GetByRefreshHash(ctx context.Context, hash string) (*model.Session, error) {
	row := db(ctx, r.pool).QueryRow(ctx, `
		SELECT id, user_id, refresh_hash, user_agent, ip::text,
		       expires_at, created_at, revoked_at
		FROM sessions WHERE refresh_hash = $1`, hash)
	return scanSession(row)
}

func (r *sessionRepository) ListActiveForUser(ctx context.Context, userID uuid.UUID) ([]*model.Session, error) {
	rows, err := db(ctx, r.pool).Query(ctx, `
		SELECT id, user_id, refresh_hash, user_agent, ip::text,
		       expires_at, created_at, revoked_at
		FROM sessions
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > now()
		ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list active sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*model.Session
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

func (r *sessionRepository) Revoke(ctx context.Context, id uuid.UUID, revokedAt time.Time) error {
	_, err := db(ctx, r.pool).Exec(ctx,
		`UPDATE sessions SET revoked_at = $1 WHERE id = $2 AND revoked_at IS NULL`,
		revokedAt, id,
	)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

func (r *sessionRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID, revokedAt time.Time) error {
	_, err := db(ctx, r.pool).Exec(ctx,
		`UPDATE sessions SET revoked_at = $1 WHERE user_id = $2 AND revoked_at IS NULL`,
		revokedAt, userID,
	)
	if err != nil {
		return fmt.Errorf("revoke all sessions for user %s: %w", userID, err)
	}
	return nil
}

func (r *sessionRepository) RevokeAllExcept(ctx context.Context, userID, keepID uuid.UUID, revokedAt time.Time) error {
	_, err := db(ctx, r.pool).Exec(ctx,
		`UPDATE sessions SET revoked_at = $1 WHERE user_id = $2 AND id <> $3 AND revoked_at IS NULL`,
		revokedAt, userID, keepID,
	)
	if err != nil {
		return fmt.Errorf("revoke sessions except %s: %w", keepID, err)
	}
	return nil
}

// scanSession works for both pgx.Row and pgx.Rows (both implement the Scan method).
func scanSession(row interface{ Scan(...any) error }) (*model.Session, error) {
	var s model.Session
	var ip *string
	err := row.Scan(
		&s.ID, &s.UserID, &s.RefreshHash, &s.UserAgent, &ip,
		&s.ExpiresAt, &s.CreatedAt, &s.RevokedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("scan session: %w", err)
	}
	if ip != nil {
		s.IP = *ip
	}
	return &s, nil
}
