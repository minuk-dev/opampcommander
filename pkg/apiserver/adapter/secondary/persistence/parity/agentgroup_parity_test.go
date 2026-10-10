package parity_test

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/inmemory"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/mongodb"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/selector"
)

// TestParity_AgentGroup reuses the shared namespaced/soft-delete contract for the
// agent-group port. PutAgentGroup takes namespace and name separately, so the put
// closure forwards them from the group's own metadata.
func TestParity_AgentGroup(t *testing.T) {
	t.Parallel()
	runAggregateParity(t, agentGroupAggregate())
}

func agentGroupAggregate() aggregate[*agentmodel.AgentGroup] {
	// ListAgentGroups is not namespace-scoped, so filter client-side to keep the
	// shared contract's namespace assertions uniform across aggregates.
	filterNS := func(items []*agentmodel.AgentGroup, ns string) []*agentmodel.AgentGroup {
		out := make([]*agentmodel.AgentGroup, 0, len(items))
		for _, it := range items {
			if it.Metadata.Namespace == ns {
				out = append(out, it)
			}
		}

		return out
	}

	putVia := func(
		put func(ctx context.Context, ns, name string, g *agentmodel.AgentGroup) (*agentmodel.AgentGroup, error),
	) func(ctx context.Context, g *agentmodel.AgentGroup) (*agentmodel.AgentGroup, error) {
		return func(ctx context.Context, g *agentmodel.AgentGroup) (*agentmodel.AgentGroup, error) {
			return put(ctx, g.Metadata.Namespace, g.Metadata.Name, g)
		}
	}

	return aggregate[*agentmodel.AgentGroup]{
		label: "agentgroup",
		makeObj: func(ns, name string) *agentmodel.AgentGroup {
			return agentmodel.NewAgentGroup(ns, name, nil, contractTime(), "tester")
		},
		setMarker:   func(g *agentmodel.AgentGroup, m string) { g.Metadata.Attributes = agentmodel.Attributes{"marker": m} },
		getMarker:   func(g *agentmodel.AgentGroup) string { return g.Metadata.Attributes["marker"] },
		markDeleted: func(g *agentmodel.AgentGroup) { g.MarkDeleted(contractTime().Add(time.Hour), "tester") },
		inmemory: func() backend[*agentmodel.AgentGroup] {
			repo := inmemory.NewAgentGroupRepository(inmemory.NewAgentRepository())

			return backend[*agentmodel.AgentGroup]{
				name: "inmemory",
				put:  putVia(repo.PutAgentGroup),
				get: func(ctx context.Context, ns, name string, incl bool) (*agentmodel.AgentGroup, error) {
					return repo.GetAgentGroup(ctx, ns, name, getOptions(incl))
				},
				listNamespace: func(ctx context.Context, ns string) ([]*agentmodel.AgentGroup, error) {
					resp, err := repo.ListAllAgentGroups(ctx, nil)
					if err != nil {
						return nil, fmt.Errorf("parity list: %w", err)
					}

					return filterNS(resp.Items, ns), nil
				},
			}
		},
		mongo: func(db *mongo.Database) backend[*agentmodel.AgentGroup] {
			repo := mongodb.NewAgentGroupRepository(db, slog.Default())

			return backend[*agentmodel.AgentGroup]{
				name: "mongodb",
				put:  putVia(repo.PutAgentGroup),
				get: func(ctx context.Context, ns, name string, incl bool) (*agentmodel.AgentGroup, error) {
					return repo.GetAgentGroup(ctx, ns, name, getOptions(incl))
				},
				listNamespace: func(ctx context.Context, ns string) ([]*agentmodel.AgentGroup, error) {
					resp, err := repo.ListAllAgentGroups(ctx, nil)
					if err != nil {
						return nil, fmt.Errorf("parity list: %w", err)
					}

					return filterNS(resp.Items, ns), nil
				},
			}
		},
	}
}

// groupStatsBackend is one backend wiring for the statistics-parity test: an
// agent repository feeding an agent-group repository over the same store.
type groupStatsBackend struct {
	name     string
	putAgent func(ctx context.Context, agent *agentmodel.Agent) error
	putGroup func(ctx context.Context, group *agentmodel.AgentGroup) (*agentmodel.AgentGroup, error)
}

// TestParity_AgentGroup_Statistics pins that both backends aggregate the same
// group membership statistics from the agent store on write. Statistics are
// computed at PutAgentGroup time by scanning agents that match the selector, so
// a divergence here would surface as wrong connected/healthy counts in the UI.
func TestParity_AgentGroup_Statistics(t *testing.T) {
	t.Parallel()

	runGroupStatistics(t, inmemoryGroupStatsBackend())

	if testing.Short() {
		return
	}

	testcontainers.SkipIfProviderIsNotHealthy(t)
	runGroupStatistics(t, mongoGroupStatsBackend(startMongoDatabase(t)))
}

func inmemoryGroupStatsBackend() groupStatsBackend {
	agentRepo := inmemory.NewAgentRepository()
	groupRepo := inmemory.NewAgentGroupRepository(agentRepo)

	return groupStatsBackend{
		name:     "inmemory",
		putAgent: agentRepo.PutAgent,
		putGroup: func(ctx context.Context, group *agentmodel.AgentGroup) (*agentmodel.AgentGroup, error) {
			return groupRepo.PutAgentGroup(ctx, group.Metadata.Namespace, group.Metadata.Name, group)
		},
	}
}

func mongoGroupStatsBackend(db *mongo.Database) groupStatsBackend {
	agentRepo := mongodb.NewAgentRepository(db, slog.Default())
	groupRepo := mongodb.NewAgentGroupRepository(db, slog.Default())

	return groupStatsBackend{
		name:     "mongodb",
		putAgent: agentRepo.PutAgent,
		putGroup: func(ctx context.Context, group *agentmodel.AgentGroup) (*agentmodel.AgentGroup, error) {
			return groupRepo.PutAgentGroup(ctx, group.Metadata.Namespace, group.Metadata.Name, group)
		},
	}
}

func runGroupStatistics(t *testing.T, b groupStatsBackend) {
	t.Helper()

	t.Run(b.name, func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		ns := uniqueNamespace("stats")
		// A per-run selector value keeps matches isolated within a shared store.
		serviceName := "otelcol-" + uuid.NewString()[:8]
		selector := map[string]string{"service.name": serviceName}

		// Connected + healthy, matching the selector.
		connected := agentmodel.NewAgent(uuid.New())
		connected.Metadata.Namespace = ns
		connected.Metadata.Description.IdentifyingAttributes = selector
		connected.Status.Connected = true
		connected.Status.LastReportedAt = time.Now()
		connected.Status.ComponentHealth.Healthy = true
		require.NoError(t, b.putAgent(ctx, connected))

		// Matching but disconnected.
		disconnected := agentmodel.NewAgent(uuid.New())
		disconnected.Metadata.Namespace = ns
		disconnected.Metadata.Description.IdentifyingAttributes = selector
		require.NoError(t, b.putAgent(ctx, disconnected))

		// Non-matching agent must not be counted.
		other := agentmodel.NewAgent(uuid.New())
		other.Metadata.Namespace = ns
		other.Metadata.Description.IdentifyingAttributes = map[string]string{"service.name": "nginx"}
		other.Status.Connected = true
		other.Status.LastReportedAt = time.Now()
		require.NoError(t, b.putAgent(ctx, other))

		group := agentmodel.NewAgentGroup(ns, "grp", nil, contractTime(), "tester")
		group.Spec.Selector = agentmodel.AgentSelector{IdentifyingAttributes: selector}

		stored, err := b.putGroup(ctx, group)
		require.NoError(t, err)

		assert.Equal(t, 2, stored.Status.NumAgents, "both matching agents counted")
		assert.Equal(t, 1, stored.Status.NumConnectedAgents)
		assert.Equal(t, 1, stored.Status.NumHealthyAgents)
		assert.Equal(t, 1, stored.Status.NumNotConnectedAgents)

		stored.Spec.Selector.IdentifyingAttributes = map[string]string{"service.name": serviceName + "-missing"}
		updated, err := b.putGroup(ctx, stored)
		require.NoError(t, err)
		assert.Equal(t, int64(2), updated.Metadata.ResourceVersion)
		assert.Zero(t, updated.Status.NumAgents, "updated selector must replace the previous statistics")
		assert.Zero(t, updated.Status.NumConnectedAgents)
		assert.Zero(t, updated.Status.NumHealthyAgents)
		assert.Zero(t, updated.Status.NumNotConnectedAgents)
	})
}

func TestParity_AgentGroupMembership(t *testing.T) {
	t.Parallel()
	t.Run("inmemory", func(t *testing.T) {
		t.Parallel()
		runMembershipPages(t, inmemory.NewAgentGroupRepository(inmemory.NewAgentRepository()))
	})

	if testing.Short() {
		return
	}

	// Non-short runs require MongoDB so CI cannot silently skip adapter parity.
	t.Run("mongodb", func(t *testing.T) {
		t.Parallel()
		runMembershipPages(t, mongodb.NewAgentGroupRepository(startMongoDatabase(t), slog.Default()))
	})
}

func runMembershipPages(t *testing.T, repo agentport.AgentGroupPersistencePort) {
	t.Helper()
	ctx := t.Context()
	ns := uniqueNamespace("membership")
	agent := agentmodel.NewAgent(uuid.New())
	agent.Metadata.Namespace = ns
	agent.Metadata.Description.IdentifyingAttributes = map[string]string{"service.name": "$otel", "empty": ""}
	agent.Metadata.Description.NonIdentifyingAttributes = map[string]string{"host.name": "node"}

	var expected []string

	for i := range 123 {
		name := fmt.Sprintf("group-%03d", i)
		group := agentmodel.NewAgentGroup(ns, name, agentmodel.Attributes{"env": "prod"}, contractTime(), "tester")
		group.Spec.Priority = 123 - i // Membership pagination uses insertion order, not config priority.

		group.Spec.Selector = agentmodel.AgentSelector{
			IdentifyingAttributes:    map[string]string{"service.name": "$otel", "empty": ""},
			NonIdentifyingAttributes: map[string]string{"host.name": "node"},
		}
		if i%3 == 0 {
			group.Spec.Selector.IdentifyingAttributes["missing"] = ""
		} else {
			expected = append(expected, name)
		}

		_, err := repo.PutAgentGroup(ctx, ns, name, group)
		require.NoError(t, err)
	}

	for _, test := range []struct {
		name      string
		selector  agentmodel.AgentSelector
		namespace string
		deleted   bool
	}{
		{name: "other-namespace", namespace: ns + "-other"},
		{name: "deleted", namespace: ns, deleted: true},
		{name: "non-identifying-mismatch", namespace: ns,
			selector: agentmodel.AgentSelector{NonIdentifyingAttributes: map[string]string{"host.name": "other"}}},
		{name: "wrong-kind", namespace: ns,
			selector: agentmodel.AgentSelector{NonIdentifyingAttributes: map[string]string{"service.name": "$otel"}}},
	} {
		group := agentmodel.NewAgentGroup(test.namespace, test.name, nil, contractTime(), "tester")

		group.Spec.Selector = test.selector
		if test.deleted {
			group.MarkDeleted(contractTime(), "tester")
		}

		_, err := repo.PutAgentGroup(ctx, test.namespace, test.name, group)
		require.NoError(t, err)
	}

	actual := make([]string, 0, len(expected))

	options := &model.ListOptions{Limit: 50}
	first, err := repo.ListAgentGroupsForAgent(ctx, agent, options)
	require.NoError(t, err)
	require.Len(t, first.Items, 50)
	require.EqualValues(t, 32, first.RemainingItemCount)

	for _, group := range first.Items {
		actual = append(actual, group.Metadata.Name)
	}
	// Soft-deleting the cursor anchor must not invalidate its sort position.
	anchor := first.Items[len(first.Items)-1]
	anchor.MarkDeleted(contractTime(), "tester")
	_, err = repo.PutAgentGroup(ctx, ns, anchor.Metadata.Name, anchor)
	require.NoError(t, err)

	options.Continue = first.Continue
	second, err := repo.ListAgentGroupsForAgent(ctx, agent, options)
	require.NoError(t, err)
	require.Len(t, second.Items, 32)
	require.Zero(t, second.RemainingItemCount)

	for _, group := range second.Items {
		actual = append(actual, group.Metadata.Name)
	}

	require.Equal(t, expected, actual)

	options.Continue = second.Continue
	end, err := repo.ListAgentGroupsForAgent(ctx, agent, options)
	require.NoError(t, err)
	require.Empty(t, end.Items)
	require.Empty(t, end.Continue)
	// Selectors and name filtering narrow the set before it is counted/paged.
	labels, err := selector.ParseLabels("env=prod")
	require.NoError(t, err)
	fields, err := selector.ParseFields("metadata.namespace=" + ns)
	require.NoError(t, err)
	filtered, err := repo.ListAgentGroupsForAgent(ctx, agent, &model.ListOptions{
		Limit: 2, NamePrefix: "group-00", NameContains: "GROUP", LabelSelector: labels, FieldSelector: fields,
	})
	require.NoError(t, err)
	require.Len(t, filtered.Items, 2)
	require.EqualValues(t, 4, filtered.RemainingItemCount)
	require.Equal(t, "group-001", filtered.Items[0].Metadata.Name)
	// Empty selectors (nil maps) match, and includeDeleted applies to groups.
	deleted, err := repo.ListAgentGroupsForAgent(ctx, agent, &model.ListOptions{
		NamePrefix: "deleted", IncludeDeleted: true,
	})
	require.NoError(t, err)
	require.Len(t, deleted.Items, 1)

	for _, limit := range []int64{-1, 1001} {
		_, err := repo.ListAgentGroupsForAgent(ctx, agent, &model.ListOptions{Limit: limit})
		require.ErrorIs(t, err, model.ErrInvalidArgument)
	}

	_, err = repo.ListAgentGroupsForAgent(ctx, agent, &model.ListOptions{Continue: "not-a-cursor"})
	require.ErrorIs(t, err, model.ErrInvalidArgument)

	agent.Metadata.Namespace = ""
	_, err = repo.ListAgentGroupsForAgent(ctx, agent, nil)
	require.ErrorIs(t, err, model.ErrInvalidArgument)
}
