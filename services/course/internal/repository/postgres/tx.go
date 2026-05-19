package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txRunner struct{ pool *pgxpool.Pool }

func NewTxRunner(pool *pgxpool.Pool) *txRunner {
	return &txRunner{pool: pool}
}

func (r *txRunner) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	return pgx.BeginTxFunc(ctx, r.pool, pgx.TxOptions{IsoLevel: pgx.ReadCommitted}, func(tx pgx.Tx) error {
		return fn(context.WithValue(ctx, ctxTxKey{}, tx))
	})
}
