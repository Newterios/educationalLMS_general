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

type verificationCodeRepository struct{ pool *pgxpool.Pool }

func NewVerificationCodeRepository(pool *pgxpool.Pool) *verificationCodeRepository {
	return &verificationCodeRepository{pool: pool}
}

func (r *verificationCodeRepository) Create(ctx context.Context, c *model.VerificationCode) error {
	_, err := db(ctx, r.pool).Exec(ctx, `
		INSERT INTO verification_codes (id, user_id, kind, code_hash, expires_at)
		VALUES ($1,$2,$3,$4,$5)`,
		c.ID, c.UserID, c.Kind, c.CodeHash, c.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("create verification code: %w", err)
	}
	return nil
}

func (r *verificationCodeRepository) GetByCodeHash(ctx context.Context, codeHash string) (*model.VerificationCode, error) {
	row := db(ctx, r.pool).QueryRow(ctx, `
		SELECT id, user_id, kind, code_hash, expires_at, used_at
		FROM verification_codes
		WHERE code_hash = $1 AND used_at IS NULL
		LIMIT 1`, codeHash)

	var c model.VerificationCode
	err := row.Scan(&c.ID, &c.UserID, &c.Kind, &c.CodeHash, &c.ExpiresAt, &c.UsedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("get verification code: %w", err)
	}
	return &c, nil
}

func (r *verificationCodeRepository) MarkUsed(ctx context.Context, id uuid.UUID, usedAt time.Time) error {
	_, err := db(ctx, r.pool).Exec(ctx,
		`UPDATE verification_codes SET used_at = $1 WHERE id = $2`,
		usedAt, id,
	)
	if err != nil {
		return fmt.Errorf("mark code used: %w", err)
	}
	return nil
}
