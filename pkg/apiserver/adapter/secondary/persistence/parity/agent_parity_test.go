package parity_test

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/inmemory"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/mongodb"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// agentBackend reduces one Agent repository implementation to the operations the
// agent parity contract needs. Agents are keyed by instance UID, are hard-deleted
// (no soft delete), and carry optimistic-concurrency versions — so this is a
// dedicated contract rather than the shared namespaced/soft-delete one.
type agentBackend struct {
	name          string
	put           func(ctx context.Context, agent *agentmodel.Agent) error
	get           func(ctx context.Context, uid uuid.UUID) (*agentmodel.Agent, error)
	del           func(ctx context.Context, uid uuid.UUID) error
	listNamespace func(ctx context.Context, ns string) ([]*agentmodel.Agent, error)
}

func makeAgent(ns, marker string) *agentmodel.Agent {
	agent := agentmodel.NewAgent(uuid.New())
	agent.Metadata.Namespace = ns
	agent.Metadata.Description.IdentifyingAttributes = map[string]string{"marker": marker}

	return agent
}

func agentMarker(agent *agentmodel.Agent) string {
	return agent.Metadata.Description.IdentifyingAttributes["marker"]
}

func setAgentMarker(agent *agentmodel.Agent, marker string) {
	agent.Metadata.Description.IdentifyingAttributes = map[string]string{"marker": marker}
}

// TestParity_Agent runs the agent-specific contract against both backends.
func TestParity_Agent(t *testing.T) {
	t.Parallel()

	runAgentContract(t, inmemoryAgentBackend())

	if testing.Short() {
		return
	}

	testcontainers.SkipIfProviderIsNotHealthy(t)
	runAgentContract(t, mongoAgentBackend(startMongoDatabase(t)))
}

func inmemoryAgentBackend() agentBackend {
	repo := inmemory.NewAgentRepository()

	return agentBackend{
		name: "inmemory",
		put:  repo.PutAgent,
		get:  repo.GetAgent,
		del:  repo.DeleteAgent,
		listNamespace: func(ctx context.Context, ns string) ([]*agentmodel.Agent, error) {
			resp, err := repo.ListAgents(ctx, ns, nil)
			if err != nil {
				return nil, fmt.Errorf("parity list: %w", err)
			}

			return resp.Items, nil
		},
	}
}

func mongoAgentBackend(db *mongo.Database) agentBackend {
	repo := mongodb.NewAgentRepository(db, slog.Default())

	return agentBackend{
		name: "mongodb",
		put:  repo.PutAgent,
		get:  repo.GetAgent,
		del:  repo.DeleteAgent,
		listNamespace: func(ctx context.Context, ns string) ([]*agentmodel.Agent, error) {
			resp, err := repo.ListAgents(ctx, ns, nil)
			if err != nil {
				return nil, fmt.Errorf("parity list: %w", err)
			}

			return resp.Items, nil
		},
	}
}

//nolint:thelper // subtest bodies are the assertions themselves, not helpers
func runAgentContract(t *testing.T, b agentBackend) {
	ctx := context.Background()

	t.Run("agent/"+b.name+"/put_get_roundtrip", func(t *testing.T) {
		t.Parallel()

		agent := makeAgent(uniqueNamespace("rt"), "v1")
		require.NoError(t, b.put(ctx, agent))

		got, err := b.get(ctx, agent.Metadata.InstanceUID)
		require.NoError(t, err)
		assert.Equal(t, "v1", agentMarker(got), "stored marker must round-trip")
	})

	t.Run("agent/"+b.name+"/get_missing_is_not_found", func(t *testing.T) {
		t.Parallel()

		_, err := b.get(ctx, uuid.New())
		require.ErrorIs(t, err, model.ErrResourceNotExist)
	})

	t.Run("agent/"+b.name+"/delete_removes_then_not_found", func(t *testing.T) {
		t.Parallel()

		agent := makeAgent(uniqueNamespace("del"), "v1")
		require.NoError(t, b.put(ctx, agent))

		require.NoError(t, b.del(ctx, agent.Metadata.InstanceUID))

		_, err := b.get(ctx, agent.Metadata.InstanceUID)
		require.ErrorIs(t, err, model.ErrResourceNotExist)

		// Deleting a missing agent reports not-found.
		require.ErrorIs(t, b.del(ctx, agent.Metadata.InstanceUID), model.ErrResourceNotExist)
	})

	t.Run("agent/"+b.name+"/namespace_isolation", func(t *testing.T) {
		t.Parallel()

		nsA, nsB := uniqueNamespace("iso-a"), uniqueNamespace("iso-b")

		inA := makeAgent(nsA, "from-a")
		require.NoError(t, b.put(ctx, inA))
		require.NoError(t, b.put(ctx, makeAgent(nsB, "from-b")))

		listA, err := b.listNamespace(ctx, nsA)
		require.NoError(t, err)
		require.Len(t, listA, 1)
		assert.Equal(t, inA.Metadata.InstanceUID, listA[0].Metadata.InstanceUID)
	})

	t.Run("agent/"+b.name+"/get_returns_isolated_copy", func(t *testing.T) {
		t.Parallel()

		agent := makeAgent(uniqueNamespace("copy"), "original")
		require.NoError(t, b.put(ctx, agent))

		got, err := b.get(ctx, agent.Metadata.InstanceUID)
		require.NoError(t, err)

		// Mutating a Get result must not leak into the store.
		setAgentMarker(got, "mutated")
		got.Status.Connected = true

		reread, err := b.get(ctx, agent.Metadata.InstanceUID)
		require.NoError(t, err)
		assert.Equal(t, "original", agentMarker(reread), "mutation must not leak into the store")
		assert.False(t, reread.Status.Connected)
	})

	t.Run("agent/"+b.name+"/optimistic_concurrency", func(t *testing.T) {
		t.Parallel()

		agent := makeAgent(uniqueNamespace("occ"), "v1")
		require.Equal(t, int64(0), agent.Metadata.ResourceVersion)

		require.NoError(t, b.put(ctx, agent))
		assert.Equal(t, int64(1), agent.Metadata.ResourceVersion, "first put inserts at v1 and bumps the caller")

		loadA, err := b.get(ctx, agent.Metadata.InstanceUID)
		require.NoError(t, err)
		loadB, err := b.get(ctx, agent.Metadata.InstanceUID)
		require.NoError(t, err)

		// Writer A wins, advancing the stored version to v2.
		setAgentMarker(loadA, "from-a")
		require.NoError(t, b.put(ctx, loadA))

		// Writer B still holds v1, so its write is rejected rather than clobbering A.
		setAgentMarker(loadB, "from-b")
		require.ErrorIs(t, b.put(ctx, loadB), model.ErrConflict)

		stored, err := b.get(ctx, agent.Metadata.InstanceUID)
		require.NoError(t, err)
		assert.Equal(t, "from-a", agentMarker(stored), "the losing writer must not overwrite the winner")
		assert.Equal(t, int64(2), stored.Metadata.ResourceVersion)
	})
}
