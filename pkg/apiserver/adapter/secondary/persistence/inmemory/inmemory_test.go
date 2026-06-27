package inmemory_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/inmemory"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// errSentinel is a static error used to assert transaction error propagation.
var errSentinel = errors.New("boom")

func TestAgentRepository_PutGetDelete(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := inmemory.NewAgentRepository()
	uid := uuid.New()
	agent := agentmodel.NewAgent(uid)

	require.NoError(t, repo.PutAgent(ctx, agent))

	got, err := repo.GetAgent(ctx, uid)
	require.NoError(t, err)
	assert.Equal(t, uid, got.Metadata.InstanceUID)

	require.NoError(t, repo.DeleteAgent(ctx, uid))

	_, err = repo.GetAgent(ctx, uid)
	require.ErrorIs(t, err, model.ErrResourceNotExist)

	// Deleting a missing agent reports not-found.
	require.ErrorIs(t, repo.DeleteAgent(ctx, uid), model.ErrResourceNotExist)
}

func TestAgentRepository_ListByNamespaceAndPagination(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := inmemory.NewAgentRepository()

	for range 3 {
		agent := agentmodel.NewAgent(uuid.New())
		agent.Metadata.Namespace = "ns-a"
		require.NoError(t, repo.PutAgent(ctx, agent))
	}

	other := agentmodel.NewAgent(uuid.New())
	other.Metadata.Namespace = "ns-b"
	require.NoError(t, repo.PutAgent(ctx, other))

	// First page of 2 from ns-a, then resume via the continue token.
	//exhaustruct:ignore
	page1, err := repo.ListAgents(ctx, "ns-a", &model.ListOptions{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, page1.Items, 2)
	assert.Equal(t, int64(1), page1.RemainingItemCount)
	require.NotEmpty(t, page1.Continue)

	//exhaustruct:ignore
	page2, err := repo.ListAgents(ctx, "ns-a", &model.ListOptions{Limit: 2, Continue: page1.Continue})
	require.NoError(t, err)
	assert.Len(t, page2.Items, 1)
	assert.Equal(t, int64(0), page2.RemainingItemCount)
}

func TestAgentRepository_SearchByInstanceUIDPrefix(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := inmemory.NewAgentRepository()

	uid := uuid.New()
	agent := agentmodel.NewAgent(uid)
	require.NoError(t, repo.PutAgent(ctx, agent))

	prefix := uid.String()[:8]

	resp, err := repo.SearchAgents(ctx, agentmodel.DefaultNamespaceName, prefix, nil)
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)
	assert.Equal(t, uid, resp.Items[0].Metadata.InstanceUID)

	// A non-matching prefix yields nothing.
	resp, err = repo.SearchAgents(ctx, agentmodel.DefaultNamespaceName, "zzzzzzzz", nil)
	require.NoError(t, err)
	assert.Empty(t, resp.Items)

	// Empty query is valid and returns an empty page.
	resp, err = repo.SearchAgents(ctx, agentmodel.DefaultNamespaceName, "", nil)
	require.NoError(t, err)
	assert.Empty(t, resp.Items)
}

func TestAgentRepository_ListByIdentifyingAttributesSelector(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := inmemory.NewAgentRepository()

	const selectorNamespace = "sel-ns"

	match := agentmodel.NewAgent(uuid.New())
	match.Metadata.Namespace = selectorNamespace
	match.Metadata.Description.IdentifyingAttributes = map[string]string{
		"service.name":      "otel-collector",
		"service.namespace": "prod",
	}
	require.NoError(t, repo.PutAgent(ctx, match))

	// Same key, different value -> excluded by an exact-match selector.
	otherValue := agentmodel.NewAgent(uuid.New())
	otherValue.Metadata.Namespace = selectorNamespace
	otherValue.Metadata.Description.IdentifyingAttributes = map[string]string{"service.name": "nginx"}
	require.NoError(t, repo.PutAgent(ctx, otherValue))

	// Matching attribute but a different namespace -> excluded by the namespace scope.
	otherNamespace := agentmodel.NewAgent(uuid.New())
	otherNamespace.Metadata.Namespace = "other-ns"
	otherNamespace.Metadata.Description.IdentifyingAttributes = map[string]string{"service.name": "otel-collector"}
	require.NoError(t, repo.PutAgent(ctx, otherNamespace))

	//exhaustruct:ignore
	resp, err := repo.ListAgents(ctx, selectorNamespace, &model.ListOptions{
		IdentifyingAttributes: map[string]string{"service.name": "otel-collector"},
	})
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)
	assert.Equal(t, match.Metadata.InstanceUID, resp.Items[0].Metadata.InstanceUID)

	// Every pair must match (AND semantics).
	//exhaustruct:ignore
	resp, err = repo.ListAgents(ctx, selectorNamespace, &model.ListOptions{
		IdentifyingAttributes: map[string]string{"service.name": "otel-collector", "service.namespace": "staging"},
	})
	require.NoError(t, err)
	assert.Empty(t, resp.Items)
}

func TestAgentRepository_ListByNonIdentifyingAttributesSelector(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := inmemory.NewAgentRepository()

	const selectorNamespace = "sel-ns"

	match := agentmodel.NewAgent(uuid.New())
	match.Metadata.Namespace = selectorNamespace
	match.Metadata.Description.IdentifyingAttributes = map[string]string{"service.name": "otel-collector"}
	match.Metadata.Description.NonIdentifyingAttributes = map[string]string{"os.type": "linux", "host.arch": "amd64"}
	require.NoError(t, repo.PutAgent(ctx, match))

	// Same non-identifying key, different value -> excluded by exact match.
	otherValue := agentmodel.NewAgent(uuid.New())
	otherValue.Metadata.Namespace = selectorNamespace
	otherValue.Metadata.Description.NonIdentifyingAttributes = map[string]string{"os.type": "windows"}
	require.NoError(t, repo.PutAgent(ctx, otherValue))

	//exhaustruct:ignore
	resp, err := repo.ListAgents(ctx, selectorNamespace, &model.ListOptions{
		NonIdentifyingAttributes: map[string]string{"os.type": "linux"},
	})
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)
	assert.Equal(t, match.Metadata.InstanceUID, resp.Items[0].Metadata.InstanceUID)

	// Identifying and non-identifying selectors are AND-combined.
	//exhaustruct:ignore
	resp, err = repo.ListAgents(ctx, selectorNamespace, &model.ListOptions{
		IdentifyingAttributes:    map[string]string{"service.name": "otel-collector"},
		NonIdentifyingAttributes: map[string]string{"os.type": "linux"},
	})
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)
	assert.Equal(t, match.Metadata.InstanceUID, resp.Items[0].Metadata.InstanceUID)

	// A non-identifying mismatch excludes the agent even when identifying matches.
	//exhaustruct:ignore
	resp, err = repo.ListAgents(ctx, selectorNamespace, &model.ListOptions{
		IdentifyingAttributes:    map[string]string{"service.name": "otel-collector"},
		NonIdentifyingAttributes: map[string]string{"os.type": "darwin"},
	})
	require.NoError(t, err)
	assert.Empty(t, resp.Items)
}

func TestNamespaceRepository_SoftDeleteHiddenUnlessIncluded(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := inmemory.NewNamespaceRepository()

	ns := agentmodel.NewNamespace("ns-soft")
	ns.MarkAsCreated(time.Now(), "tester")
	_, err := repo.PutNamespace(ctx, ns)
	require.NoError(t, err)

	ns.MarkAsDeleted(time.Now(), "tester")
	_, err = repo.PutNamespace(ctx, ns)
	require.NoError(t, err)

	// Default read excludes the soft-deleted namespace.
	_, err = repo.GetNamespace(ctx, "ns-soft", nil)
	require.ErrorIs(t, err, model.ErrResourceNotExist)

	// IncludeDeleted surfaces it again.
	//exhaustruct:ignore
	got, err := repo.GetNamespace(ctx, "ns-soft", &model.GetOptions{IncludeDeleted: true})
	require.NoError(t, err)
	assert.Equal(t, "ns-soft", got.Metadata.Name)

	// It is excluded from the default listing.
	list, err := repo.ListNamespaces(ctx, nil)
	require.NoError(t, err)
	assert.Empty(t, list.Items)
}

func TestAgentGroupRepository_StatisticsFromAgentStore(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	agentRepo := inmemory.NewAgentRepository()
	groupRepo := inmemory.NewAgentGroupRepository(agentRepo)

	selector := map[string]string{"service.name": "otelcol"}

	// A connected+healthy agent matching the selector.
	connected := agentmodel.NewAgent(uuid.New())
	connected.Metadata.Description.IdentifyingAttributes = selector
	connected.Status.Connected = true
	connected.Status.LastReportedAt = time.Now()
	connected.Status.ComponentHealth.Healthy = true
	require.NoError(t, agentRepo.PutAgent(ctx, connected))

	// A matching but disconnected agent.
	disconnected := agentmodel.NewAgent(uuid.New())
	disconnected.Metadata.Description.IdentifyingAttributes = selector
	require.NoError(t, agentRepo.PutAgent(ctx, disconnected))

	// A non-matching agent must not be counted.
	otherAgent := agentmodel.NewAgent(uuid.New())
	otherAgent.Metadata.Description.IdentifyingAttributes = map[string]string{"service.name": "nginx"}
	otherAgent.Status.Connected = true
	otherAgent.Status.LastReportedAt = time.Now()
	require.NoError(t, agentRepo.PutAgent(ctx, otherAgent))

	group := agentmodel.NewAgentGroup("default", "grp", nil, time.Now(), "tester")
	group.Spec.Selector = agentmodel.AgentSelector{
		IdentifyingAttributes:    selector,
		NonIdentifyingAttributes: nil,
	}

	stored, err := groupRepo.PutAgentGroup(ctx, "default", "grp", group)
	require.NoError(t, err)

	assert.Equal(t, 2, stored.Status.NumAgents)
	assert.Equal(t, 1, stored.Status.NumConnectedAgents)
	assert.Equal(t, 1, stored.Status.NumHealthyAgents)
	assert.Equal(t, 0, stored.Status.NumUnhealthyAgents)
	assert.Equal(t, 1, stored.Status.NumNotConnectedAgents)
}

func TestEndpointRepository_PutGetSoftDeleteAndIsolation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := inmemory.NewEndpointRepository()

	now := time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC)
	endpoint := agentmodel.NewEndpoint("default", "tempo", nil, now, "tester")
	endpoint.Spec.URL = "https://tempo.example.com"
	endpoint.Spec.Signals = agentmodel.EndpointSignals{Metrics: false, Logs: false, Traces: true}
	endpoint.Spec.Tenants = []agentmodel.EndpointTenant{
		{Name: "team-a", Headers: map[string]string{"X-Scope-OrgID": "team-a"}, Tags: nil, Signals: nil},
	}

	_, err := repo.PutEndpoint(ctx, endpoint)
	require.NoError(t, err)

	// A same-named endpoint in another namespace must not bleed into the
	// default-namespace listing.
	other := agentmodel.NewEndpoint("other", "tempo", nil, now, "tester")
	_, err = repo.PutEndpoint(ctx, other)
	require.NoError(t, err)

	listDefault, err := repo.ListEndpoints(ctx, "default", nil)
	require.NoError(t, err)
	require.Len(t, listDefault.Items, 1)
	assert.Equal(t, "default", listDefault.Items[0].Metadata.Namespace)

	got, err := repo.GetEndpoint(ctx, "default", "tempo", nil)
	require.NoError(t, err)
	assert.Equal(t, "https://tempo.example.com", got.Spec.URL)
	require.Len(t, got.Spec.Tenants, 1)
	assert.Equal(t, "team-a", got.Spec.Tenants[0].Headers["X-Scope-OrgID"])

	// Mutating a Get result must not leak into the store (deep-copy semantics).
	got.Spec.Tenants[0].Headers["X-Scope-OrgID"] = "mutated"

	reread, err := repo.GetEndpoint(ctx, "default", "tempo", nil)
	require.NoError(t, err)
	assert.Equal(t, "team-a", reread.Spec.Tenants[0].Headers["X-Scope-OrgID"])

	// Soft delete hides it unless IncludeDeleted is set.
	endpoint.MarkDeleted(now.Add(time.Hour), "tester")
	_, err = repo.PutEndpoint(ctx, endpoint)
	require.NoError(t, err)

	_, err = repo.GetEndpoint(ctx, "default", "tempo", nil)
	require.ErrorIs(t, err, model.ErrResourceNotExist)

	//exhaustruct:ignore
	revived, err := repo.GetEndpoint(ctx, "default", "tempo", &model.GetOptions{IncludeDeleted: true})
	require.NoError(t, err)
	assert.Equal(t, "tempo", revived.Metadata.Name)

	list, err := repo.ListEndpoints(ctx, "default", nil)
	require.NoError(t, err)
	assert.Empty(t, list.Items)
}

func TestAgentRepository_GetReturnsIsolatedCopy(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := inmemory.NewAgentRepository()
	uid := uuid.New()
	require.NoError(t, repo.PutAgent(ctx, agentmodel.NewAgent(uid)))

	// Mutating the value returned by Get must not affect the stored copy, since
	// the store hands out deep copies (matching MongoDB's fresh-copy semantics).
	got, err := repo.GetAgent(ctx, uid)
	require.NoError(t, err)

	got.Status.Connected = true
	got.Metadata.Namespace = "mutated"

	reread, err := repo.GetAgent(ctx, uid)
	require.NoError(t, err)
	assert.False(t, reread.Status.Connected, "mutation of a Get result must not leak into the store")
	assert.Equal(t, agentmodel.DefaultNamespaceName, reread.Metadata.Namespace)
}

// TestAgentGroupRepository_ConcurrentAccessNoRace exercises the store under
// concurrent readers and writers; it is meaningful under `go test -race`, where
// it would previously have reported a data race on the shared stored pointer.
func TestAgentGroupRepository_ConcurrentAccessNoRace(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	agentRepo := inmemory.NewAgentRepository()
	groupRepo := inmemory.NewAgentGroupRepository(agentRepo)

	const groupName = "race-grp"

	group := agentmodel.NewAgentGroup("default", groupName, nil, time.Now(), "tester")
	_, err := groupRepo.PutAgentGroup(ctx, "default", groupName, group)
	require.NoError(t, err)

	var waitGroup sync.WaitGroup

	const workers = 8

	for range workers {
		// Each iteration starts one reader and one writer goroutine.
		waitGroup.Add(2)

		go func() {
			defer waitGroup.Done()

			for range 50 {
				_, _ = groupRepo.GetAgentGroup(ctx, "default", groupName, nil)
				_, _ = groupRepo.ListAgentGroups(ctx, nil)
			}
		}()

		go func() {
			defer waitGroup.Done()

			for range 50 {
				updated := agentmodel.NewAgentGroup("default", groupName, nil, time.Now(), "tester")
				_, _ = groupRepo.PutAgentGroup(ctx, "default", groupName, updated)
			}
		}()
	}

	waitGroup.Wait()
}

func TestTransactionRunner_RunsCallback(t *testing.T) {
	t.Parallel()

	runner := inmemory.NewTransactionRunner()

	called := false
	err := runner.WithinTransaction(context.Background(), func(context.Context) error {
		called = true

		return nil
	})
	require.NoError(t, err)
	assert.True(t, called)

	err = runner.WithinTransaction(context.Background(), func(context.Context) error {
		return errSentinel
	})
	require.ErrorIs(t, err, errSentinel)
}

func TestAgentRepository_PutAgentBumpsResourceVersion(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := inmemory.NewAgentRepository()
	uid := uuid.New()
	agent := agentmodel.NewAgent(uid)

	// A freshly created agent starts at version 0; the first write inserts it as v1.
	assert.Equal(t, int64(0), agent.Metadata.ResourceVersion)
	require.NoError(t, repo.PutAgent(ctx, agent))
	assert.Equal(t, int64(1), agent.Metadata.ResourceVersion, "PutAgent must bump the caller's version")

	got, err := repo.GetAgent(ctx, uid)
	require.NoError(t, err)
	assert.Equal(t, int64(1), got.Metadata.ResourceVersion)

	// A read-modify-write cycle advances the version again.
	got.Metadata.Namespace = "ns-x"
	require.NoError(t, repo.PutAgent(ctx, got))
	assert.Equal(t, int64(2), got.Metadata.ResourceVersion)
}

func TestAgentRepository_PutAgentConflictOnStaleVersion(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := inmemory.NewAgentRepository()
	uid := uuid.New()

	first := agentmodel.NewAgent(uid)
	require.NoError(t, repo.PutAgent(ctx, first)) // stored at v1

	// Two writers load the same v1 snapshot.
	loadA, err := repo.GetAgent(ctx, uid)
	require.NoError(t, err)
	loadB, err := repo.GetAgent(ctx, uid)
	require.NoError(t, err)

	// Writer A wins, advancing the stored version to v2.
	loadA.Metadata.Namespace = "from-a"
	require.NoError(t, repo.PutAgent(ctx, loadA))

	// Writer B still holds v1, so its write is rejected instead of clobbering A.
	loadB.Metadata.Namespace = "from-b"
	require.ErrorIs(t, repo.PutAgent(ctx, loadB), model.ErrConflict)

	stored, err := repo.GetAgent(ctx, uid)
	require.NoError(t, err)
	assert.Equal(t, "from-a", stored.Metadata.Namespace, "the losing writer must not overwrite the winner")
	assert.Equal(t, int64(2), stored.Metadata.ResourceVersion)
}

func TestAgentRepository_PutAgentConflictOnConcurrentCreate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := inmemory.NewAgentRepository()
	uid := uuid.New()

	// Two nodes independently GetOrCreate the same not-yet-persisted agent (both v0).
	createA := agentmodel.NewAgent(uid)
	createB := agentmodel.NewAgent(uid)

	require.NoError(t, repo.PutAgent(ctx, createA))
	// The second create-as-v0 must conflict rather than insert a duplicate.
	require.ErrorIs(t, repo.PutAgent(ctx, createB), model.ErrConflict)
}

func TestServerConnectionRepository_ReplaceAndList(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := inmemory.NewServerConnectionRepository()
	now := time.Now()

	sc := func(server, ns string, uid uuid.UUID) *agentmodel.ServerConnection {
		return &agentmodel.ServerConnection{
			ServerID:           server,
			UID:                uid,
			InstanceUID:        uuid.New(),
			Type:               agentmodel.ConnectionTypeWebSocket,
			Namespace:          ns,
			LastCommunicatedAt: now,
			SnapshotAt:         now,
		}
	}

	a1, a2, b1 := uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, repo.ReplaceServerConnections(ctx, "server-a", []*agentmodel.ServerConnection{
		sc("server-a", "default", a1), sc("server-a", "default", a2),
	}))
	require.NoError(t, repo.ReplaceServerConnections(ctx, "server-b", []*agentmodel.ServerConnection{
		sc("server-b", "default", b1),
	}))

	// Cluster view spans both servers.
	resp, err := repo.ListServerConnections(ctx, "default", time.Time{}, nil)
	require.NoError(t, err)
	assert.Len(t, resp.Items, 3)

	// Replacing one server's set only affects that server.
	require.NoError(t, repo.ReplaceServerConnections(ctx, "server-a", []*agentmodel.ServerConnection{
		sc("server-a", "default", a1),
	}))

	resp, err = repo.ListServerConnections(ctx, "default", time.Time{}, nil)
	require.NoError(t, err)
	assert.Len(t, resp.Items, 2) // a1 + b1

	// Clearing a server removes its records.
	require.NoError(t, repo.ReplaceServerConnections(ctx, "server-b", nil))

	resp, err = repo.ListServerConnections(ctx, "default", time.Time{}, nil)
	require.NoError(t, err)
	assert.Len(t, resp.Items, 1)
	assert.Equal(t, "server-a", resp.Items[0].ServerID)
}

func TestServerConnectionRepository_ListFiltersNamespaceAndStaleness(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := inmemory.NewServerConnectionRepository()
	now := time.Now()

	fresh := &agentmodel.ServerConnection{
		ServerID: "server-a", UID: uuid.New(), InstanceUID: uuid.New(),
		Type: agentmodel.ConnectionTypeHTTP, Namespace: "ns-a",
		LastCommunicatedAt: now, SnapshotAt: now,
	}
	otherNS := &agentmodel.ServerConnection{
		ServerID: "server-a", UID: uuid.New(), InstanceUID: uuid.New(),
		Type: agentmodel.ConnectionTypeHTTP, Namespace: "ns-b",
		LastCommunicatedAt: now, SnapshotAt: now,
	}
	stale := &agentmodel.ServerConnection{
		ServerID: "server-c", UID: uuid.New(), InstanceUID: uuid.New(),
		Type: agentmodel.ConnectionTypeHTTP, Namespace: "ns-a",
		LastCommunicatedAt: now, SnapshotAt: now.Add(-10 * time.Minute),
	}

	require.NoError(t, repo.ReplaceServerConnections(ctx, "server-a",
		[]*agentmodel.ServerConnection{fresh, otherNS}))
	require.NoError(t, repo.ReplaceServerConnections(ctx, "server-c",
		[]*agentmodel.ServerConnection{stale}))

	// Namespace filter + staleness cutoff (notBefore) excludes other-namespace and stale records.
	resp, err := repo.ListServerConnections(ctx, "ns-a", now.Add(-90*time.Second), nil)
	require.NoError(t, err)
	require.Len(t, resp.Items, 1)
	assert.Equal(t, fresh.UID, resp.Items[0].UID)
}
