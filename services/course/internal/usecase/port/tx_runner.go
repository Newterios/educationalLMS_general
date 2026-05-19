package port

import "context"

type TxRunner interface {
	WithinTx(ctx context.Context, fn func(context.Context) error) error
}
