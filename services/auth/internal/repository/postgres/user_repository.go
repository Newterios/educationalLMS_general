package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Newterios/lms-system-prob/services/auth/internal/model"
	"github.com/google/uuid"
)

type userRepository struct{ pool *pgxpool.Pool }

func NewUserRepository(pool *pgxpool.Pool) *userRepository {
	return &userRepository{pool: pool}
}

func (r *userRepository) Create(ctx context.Context, u *model.User) error {
	_, err := db(ctx, r.pool).Exec(ctx, `
		INSERT INTO users
			(id, email, password_hash, full_name, locale, role, email_verified, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		u.ID, u.Email, u.PasswordHash, u.FullName, u.Locale, u.Role,
		u.EmailVerified, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("create user: %w", model.ErrAlreadyExists)
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	row := db(ctx, r.pool).QueryRow(ctx, `
		SELECT id, email, password_hash, full_name, locale, role,
		       email_verified, created_at, updated_at
		FROM users WHERE id = $1`, id)
	return scanUser(row)
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	row := db(ctx, r.pool).QueryRow(ctx, `
		SELECT id, email, password_hash, full_name, locale, role,
		       email_verified, created_at, updated_at
		FROM users WHERE email = $1`, email)
	return scanUser(row)
}

func (r *userRepository) Update(ctx context.Context, u *model.User) error {
	tag, err := db(ctx, r.pool).Exec(ctx, `
		UPDATE users
		SET email=$1, password_hash=$2, full_name=$3, locale=$4,
		    role=$5, email_verified=$6, updated_at=$7
		WHERE id=$8`,
		u.Email, u.PasswordHash, u.FullName, u.Locale,
		u.Role, u.EmailVerified, u.UpdatedAt, u.ID,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update user %s: %w", u.ID, model.ErrNotFound)
	}
	return nil
}

func scanUser(row pgx.Row) (*model.User, error) {
	var u model.User
	err := row.Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.Locale,
		&u.Role, &u.EmailVerified, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}
	return &u, nil
}
