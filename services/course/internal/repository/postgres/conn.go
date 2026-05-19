package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ctxTxKey struct{}

type dbQuerier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func db(ctx context.Context, pool *pgxpool.Pool) dbQuerier {
	if tx, ok := ctx.Value(ctxTxKey{}).(pgx.Tx); ok {
		return tx
	}
	return pool
}
