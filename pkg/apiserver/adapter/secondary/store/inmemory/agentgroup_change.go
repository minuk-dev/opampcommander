package inmemory

import (
	"context"
	"fmt"

	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
)

var _ agentport.AgentGroupChangeStore = (*AgentGroupChangeStore)(nil)

// AgentGroupChangeStore owns the bounded queue of group identities. Queuing
// values rather than resource pointers prevents later caller mutations leaking
// into the propagation worker.
type AgentGroupChangeStore struct {
	changes chan agentport.AgentGroupChange
}

// NewAgentGroupChangeStore creates a node-local propagation queue.
func NewAgentGroupChangeStore(capacity int) *AgentGroupChangeStore {
	return &AgentGroupChangeStore{changes: make(chan agentport.AgentGroupChange, capacity)}
}

// Enqueue waits for queue capacity or cancellation.
func (s *AgentGroupChangeStore) Enqueue(ctx context.Context, change agentport.AgentGroupChange) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("enqueue agent group change: %w", ctx.Err())
	case s.changes <- change:
		return nil
	}
}

// TryEnqueue records a change without blocking the request on a full queue.
func (s *AgentGroupChangeStore) TryEnqueue(change agentport.AgentGroupChange) bool {
	select {
	case s.changes <- change:
		return true
	default:
		return false
	}
}

// Next waits for a queued identity or cancellation.
func (s *AgentGroupChangeStore) Next(ctx context.Context) (agentport.AgentGroupChange, error) {
	err := ctx.Err()
	if err != nil {
		return agentport.AgentGroupChange{}, fmt.Errorf("next agent group change: %w", err)
	}

	select {
	case <-ctx.Done():
		return agentport.AgentGroupChange{}, fmt.Errorf("next agent group change: %w", ctx.Err())
	case change := <-s.changes:
		return change, nil
	}
}
