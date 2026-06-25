package agentservice

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/reconcile"
)

// Reconcile kind selectors. These are the stable, public identifiers used by the API and
// CLI to address a reconciler; keep them lowercase to match opampctl resource names.
const (
	agentRemoteConfigKind = "agentremoteconfig"
	agentGroupKind        = "agentgroup"
	agentKind             = "agent"
)

// Compile-time checks that the adapters satisfy the generic reconcile contract.
var (
	_ reconcile.Reconciler = (*AgentRemoteConfigReconciler)(nil)
	_ reconcile.Reconciler = (*AgentGroupReconciler)(nil)
	_ reconcile.Reconciler = (*AgentReconciler)(nil)
)

// AgentRemoteConfigReconciler adapts AgentRemoteConfigUsecase to the generic reconcile
// registry: reconciling re-detects endpoints from the config and re-propagates it to groups.
type AgentRemoteConfigReconciler struct {
	usecase agentport.AgentRemoteConfigUsecase
}

// NewAgentRemoteConfigReconciler creates an AgentRemoteConfigReconciler.
func NewAgentRemoteConfigReconciler(usecase agentport.AgentRemoteConfigUsecase) *AgentRemoteConfigReconciler {
	return &AgentRemoteConfigReconciler{usecase: usecase}
}

// Kind implements reconcile.Reconciler.
func (*AgentRemoteConfigReconciler) Kind() string { return agentRemoteConfigKind }

// Reconcile implements reconcile.Reconciler.
func (r *AgentRemoteConfigReconciler) Reconcile(ctx context.Context, namespace, name string) error {
	err := r.usecase.ReconcileAgentRemoteConfig(ctx, namespace, name)
	if err != nil {
		return fmt.Errorf("reconcile agent remote config: %w", err)
	}

	return nil
}

// AgentGroupReconciler adapts AgentGroupUsecase to the generic reconcile registry:
// reconciling re-applies the group to its matching agents.
type AgentGroupReconciler struct {
	usecase agentport.AgentGroupUsecase
}

// NewAgentGroupReconciler creates an AgentGroupReconciler.
func NewAgentGroupReconciler(usecase agentport.AgentGroupUsecase) *AgentGroupReconciler {
	return &AgentGroupReconciler{usecase: usecase}
}

// Kind implements reconcile.Reconciler.
func (*AgentGroupReconciler) Kind() string { return agentGroupKind }

// Reconcile implements reconcile.Reconciler.
func (r *AgentGroupReconciler) Reconcile(ctx context.Context, namespace, name string) error {
	err := r.usecase.ReconcileAgentGroup(ctx, namespace, name)
	if err != nil {
		return fmt.Errorf("reconcile agent group: %w", err)
	}

	return nil
}

// AgentReconciler adapts the agent reconcile flow to the generic registry. The name is the
// agent's instance UID; reconciling re-applies the matching agent groups to the agent and
// persists the result. It enforces the namespace so an agent in another namespace is not
// reachable through a mismatched path.
type AgentReconciler struct {
	agentUsecase agentport.AgentUsecase
	groupUsecase agentport.AgentGroupUsecase
}

// NewAgentReconciler creates an AgentReconciler.
func NewAgentReconciler(
	agentUsecase agentport.AgentUsecase,
	groupUsecase agentport.AgentGroupUsecase,
) *AgentReconciler {
	return &AgentReconciler{agentUsecase: agentUsecase, groupUsecase: groupUsecase}
}

// Kind implements reconcile.Reconciler.
func (*AgentReconciler) Kind() string { return agentKind }

// Reconcile implements reconcile.Reconciler. name is the agent's instance UID.
func (r *AgentReconciler) Reconcile(ctx context.Context, namespace, name string) error {
	instanceUID, err := uuid.Parse(name)
	if err != nil {
		return fmt.Errorf("invalid agent instance uid %q: %w: %w", name, port.ErrInvalidArgument, err)
	}

	agent, err := r.agentUsecase.GetAgent(ctx, instanceUID)
	if err != nil {
		return fmt.Errorf("get agent: %w", err)
	}

	// Scope the agent to the requested namespace: an agent in another namespace must not be
	// reachable here. Treat a mismatch as not-found so it maps to 404 like a missing agent.
	if agent.Metadata.Namespace != namespace {
		return fmt.Errorf("agent %s not in namespace %q: %w", instanceUID, namespace, port.ErrResourceNotExist)
	}

	err = r.groupUsecase.ReconcileAgent(ctx, agent)
	if err != nil {
		return fmt.Errorf("reconcile agent: %w", err)
	}

	return nil
}
