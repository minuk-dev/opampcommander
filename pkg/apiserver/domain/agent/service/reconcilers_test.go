package agentservice_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/port"
)

// fakeAgentLoader returns a fixed agent by UID; the embedded interface covers unused methods.
type fakeAgentLoader struct {
	agentport.AgentUsecase

	agent *agentmodel.Agent
}

func (f *fakeAgentLoader) GetAgent(_ context.Context, _ uuid.UUID) (*agentmodel.Agent, error) {
	if f.agent == nil {
		return nil, port.ErrResourceNotExist
	}

	return f.agent, nil
}

// recordingGroupReconciler records whether ReconcileAgent ran.
type recordingGroupReconciler struct {
	agentport.AgentGroupUsecase

	called bool
}

func (r *recordingGroupReconciler) ReconcileAgent(_ context.Context, _ *agentmodel.Agent) error {
	r.called = true

	return nil
}

func TestAgentReconciler_Reconcile(t *testing.T) {
	t.Parallel()

	t.Run("reconciles an agent in the requested namespace", func(t *testing.T) {
		t.Parallel()

		uid := uuid.New()
		agent := agentmodel.NewAgent(uid)
		agent.Metadata.Namespace = "team-a"

		group := &recordingGroupReconciler{}
		reconciler := agentservice.NewAgentReconciler(&fakeAgentLoader{agent: agent}, group)

		err := reconciler.Reconcile(context.Background(), "team-a", uid.String())

		require.NoError(t, err)
		assert.True(t, group.called, "the agent should have been reconciled")
	})

	t.Run("treats a namespace mismatch as not found and does not reconcile", func(t *testing.T) {
		t.Parallel()

		uid := uuid.New()
		agent := agentmodel.NewAgent(uid)
		agent.Metadata.Namespace = "team-a"

		group := &recordingGroupReconciler{}
		reconciler := agentservice.NewAgentReconciler(&fakeAgentLoader{agent: agent}, group)

		err := reconciler.Reconcile(context.Background(), "team-b", uid.String())

		require.ErrorIs(t, err, port.ErrResourceNotExist)
		assert.False(t, group.called, "an agent in another namespace must not be reconciled")
	})

	t.Run("rejects a non-uuid name as an invalid argument", func(t *testing.T) {
		t.Parallel()

		group := &recordingGroupReconciler{}
		reconciler := agentservice.NewAgentReconciler(&fakeAgentLoader{agent: nil}, group)

		err := reconciler.Reconcile(context.Background(), "team-a", "not-a-uuid")

		require.ErrorIs(t, err, port.ErrInvalidArgument)
		assert.False(t, group.called)
	})
}
