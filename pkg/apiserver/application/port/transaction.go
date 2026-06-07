package port

import "context"

// TransactionRunner runs a function within a persistence transaction.
//
// Implementations stash transaction state (e.g. a database session) on the
// context passed to fn, so any repository operation performed with that
// context participates in the same transaction. Callers must thread the
// derived context through every persistence call they want to be atomic.
//
// The interface is intentionally technology-agnostic so the application
// layer does not depend on a specific database driver. Persistence adapters
// (MongoDB, Postgres, ...) provide their own implementations.
type TransactionRunner interface {
	// WithinTransaction starts a transaction, invokes fn with a derived
	// context, and commits if fn returns nil. If fn returns an error, the
	// transaction is rolled back and the error is propagated to the caller.
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
