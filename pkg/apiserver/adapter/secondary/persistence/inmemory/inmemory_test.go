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
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/port"
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
	require.ErrorIs(t, err, port.ErrResourceNotExist)

	// Deleting a missing agent reports not-found.
	require.ErrorIs(t, repo.DeleteAgent(ctx, uid), port.ErrResourceNotExist)
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
	require.ErrorIs(t, err, port.ErrResourceNotExist)

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
