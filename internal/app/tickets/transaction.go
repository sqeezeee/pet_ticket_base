package tickets

import "context"

type transactionContextKey struct{}

// ContextWithTx shares one typed key across the application and infrastructure.
func ContextWithTx(ctx context.Context, tx TxCommitter) context.Context {
	return context.WithValue(ctx, transactionContextKey{}, tx)
}

// TxFromContext returns the transaction bound to this unit of work.
func TxFromContext(ctx context.Context) TxCommitter {
	tx, _ := ctx.Value(transactionContextKey{}).(TxCommitter)
	return tx
}
