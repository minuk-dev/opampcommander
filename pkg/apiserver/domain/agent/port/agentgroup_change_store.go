package agentport

import "context"

// AgentGroupChange identifies a saved or deleted group needing propagation.
// Consumers reload it from persistence, including soft-deleted groups.
type AgentGroupChange struct {
	Namespace string
	Name      string
}

// AgentGroupChangeStore owns the node-local propagation queue. The periodic
// reconcile remains the recovery path for changes missed by this bounded queue.
type AgentGroupChangeStore interface {
	Enqueue(ctx context.Context, change AgentGroupChange) error
	TryEnqueue(change AgentGroupChange) bool
	Next(ctx context.Context) (AgentGroupChange, error)
}
