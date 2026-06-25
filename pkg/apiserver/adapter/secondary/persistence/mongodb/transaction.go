package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
)

var (
	_ port.TransactionRunner    = (*TransactionRunner)(nil)
	_ agentport.TransactionPort = (*TransactionRunner)(nil)
)

// TransactionRunner is the MongoDB-backed implementation of
// [port.TransactionRunner]. It runs the callback inside a MongoDB
// session/transaction; the driver automatically routes any collection
// operation issued with the session-bearing context through that session,
// so repositories do not need to know about transactions.
//
// MongoDB transactions require a replica set or sharded cluster. The dev
// setup runs a single-node replica set (see Makefile).
type TransactionRunner struct {
	client *mongo.Client
}

// NewTransactionRunner creates a new MongoDB TransactionRunner.
func NewTransactionRunner(client *mongo.Client) *TransactionRunner {
	return &TransactionRunner{client: client}
}

// txCallbackResult is a placeholder so we can return (any, error) from the
// mongo-driver callback without violating the nilnil lint rule.
type txCallbackResult struct{}

// WithinTransaction implements [port.TransactionRunner].
//
// callback may be invoked more than once if the driver retries on transient
// errors, so the function must be idempotent.
func (r *TransactionRunner) WithinTransaction(
	ctx context.Context,
	callback func(ctx context.Context) error,
) error {
	session, err := r.client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start mongo session: %w", err)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sessCtx context.Context) (any, error) {
		cbErr := callback(sessCtx)
		if cbErr != nil {
			return txCallbackResult{}, cbErr
		}

		return txCallbackResult{}, nil
	})
	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}
