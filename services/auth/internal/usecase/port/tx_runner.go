package port

import "context"

// TxRunner executes fn inside a single DB transaction (ReadCommitted isolation).
// The active pgx.Tx is stored in the context; repository methods detect it
// and use it instead of the pool. A non-nil error from fn triggers rollback.
// Used by: ConfirmPasswordReset, ChangePassword (Phase 1C).
type TxRunner interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}
