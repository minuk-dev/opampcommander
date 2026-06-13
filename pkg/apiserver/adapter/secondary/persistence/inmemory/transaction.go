package inmemory

import (
	"context"
	"fmt"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

var _ port.TransactionRunner = (*TransactionRunner)(nil)

// TransactionRunner is the in-memory implementation of [port.TransactionRunner].
//
// The in-memory stores are not transactional: there is no snapshot/rollback, so
// the callback simply runs against the live stores. A callback that fails after
// already mutating some stores leaves those mutations in place. This is an
// accepted limitation of standalone mode, which targets single-node development
// and demo use rather than the multi-server consistency guarantees MongoDB
// transactions provide.
type TransactionRunner struct{}

// NewTransactionRunner creates a new in-memory TransactionRunner.
func NewTransactionRunner() *TransactionRunner {
	return &TransactionRunner{}
}

// WithinTransaction implements [port.TransactionRunner] by invoking fn directly
// with the unmodified context.
func (r *TransactionRunner) WithinTransaction(
	ctx context.Context,
	callback func(ctx context.Context) error,
) error {
	err := callback(ctx)
	if err != nil {
		return fmt.Errorf("in-memory transaction callback failed: %w", err)
	}

	return nil
}
