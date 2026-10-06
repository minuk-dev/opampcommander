package agentport

import (
	"context"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
)

// AgentGroupChange identifies a saved or deleted group needing propagation.
// Consumers reload its configuration, but retain the affected selector so a later
// selector change or recreation cannot strand its former members.
type AgentGroupChange struct {
	Namespace string
	Name      string
	Selector  agentmodel.AgentSelector
}

// AgentGroupChangeStore owns the node-local propagation queue. The periodic
// reconcile remains the recovery path for changes missed by this bounded queue.
type AgentGroupChangeStore interface {
	Enqueue(ctx context.Context, change AgentGroupChange) error
	TryEnqueue(change AgentGroupChange) bool
	Next(ctx context.Context) (AgentGroupChange, error)
}
