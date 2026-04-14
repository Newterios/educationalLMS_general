package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txRunner struct{ pool *pgxpool.Pool }

// NewTxRunner wraps a pool and implements port.TxManager.
func NewTxRunner(pool *pgxpool.Pool) *txRunner { return &txRunner{pool: pool} }

// WithinTx runs fn inside a single ReadCommitted transaction.
// The transaction is stored in ctx via ctxTxKey so that all repository methods
// that call db(ctx, pool) automatically participate in the same TX.
func (r *txRunner) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	return pgx.BeginTxFunc(ctx, r.pool, pgx.TxOptions{IsoLevel: pgx.ReadCommitted}, func(tx pgx.Tx) error {
		return fn(context.WithValue(ctx, ctxTxKey{}, tx))
	})
}
