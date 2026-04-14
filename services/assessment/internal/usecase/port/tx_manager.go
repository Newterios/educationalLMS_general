package port

import "context"

// TxManager runs fn inside a single DB transaction.
// The transaction is stored in the context; repository methods detect it
// and use it instead of the pool. Used only for SubmitAttempt and
// GradeSubmission (ARCHITECTURE.md §4.3).
type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}
