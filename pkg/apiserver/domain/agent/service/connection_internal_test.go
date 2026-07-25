package agentservice

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

type stubServerIdentity struct {
	id string
}

func (s stubServerIdentity) CurrentServerID() string { return s.id }
func (s stubServerIdentity) CurrentServer(context.Context) (*agentmodel.Server, error) {
	return &agentmodel.Server{ID: s.id}, nil
}

type fakeServerConnectionStore struct {
	replacedServerID string
	replaced         []*agentmodel.ServerConnection
	listNotBefore    time.Time
	listServerID     string
	listResult       []*agentmodel.ServerConnection
}

func (f *fakeServerConnectionStore) ReplaceServerConnections(
	_ context.Context, serverID string, conns []*agentmodel.ServerConnection,
) error {
	f.replacedServerID = serverID
	f.replaced = conns

	return nil
}

func (f *fakeServerConnectionStore) ListServerConnections(
	_ context.Context, _ string, serverID string, notBefore time.Time, _ *model.ListOptions,
) (*model.ListResponse[*agentmodel.ServerConnection], error) {
	f.listNotBefore = notBefore
	f.listServerID = serverID

	return &model.ListResponse[*agentmodel.ServerConnection]{
		Items:              f.listResult,
		Continue:           "",
		RemainingItemCount: 0,
	}, nil
}

func TestConnectionService_snapshotConnections(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := &fakeServerConnectionStore{}
	svc := NewConnectionService(nil, stubServerIdentity{id: "server-1"}, store, slog.Default())

	instanceUID := uuid.New()
	conn := agentmodel.NewConnection("conn-key", agentmodel.ConnectionTypeWebSocket)
	conn.SetInstanceUID(instanceUID)
	conn.SetNamespace("default")
	require.NoError(t, svc.SaveConnection(ctx, conn))

	svc.snapshotConnections(ctx)

	assert.Equal(t, "server-1", store.replacedServerID)
	require.Len(t, store.replaced, 1)
	assert.Equal(t, "server-1", store.replaced[0].ServerID)
	assert.Equal(t, instanceUID, store.replaced[0].InstanceUID)
	assert.Equal(t, conn.UID, store.replaced[0].UID)
}

func TestConnectionService_snapshotConnectionsSkipsWithoutIdentity(t *testing.T) {
	t.Parallel()

	store := &fakeServerConnectionStore{}
	svc := NewConnectionService(nil, stubServerIdentity{id: ""}, store, slog.Default())

	svc.snapshotConnections(context.Background())

	assert.Empty(t, store.replacedServerID)
	assert.Nil(t, store.replaced)
}

func TestConnectionService_ListClusterConnectionsAppliesStalenessWindow(t *testing.T) {
	t.Parallel()

	store := &fakeServerConnectionStore{
		listResult: []*agentmodel.ServerConnection{
			{ServerID: "server-2", UID: uuid.New(), Namespace: "default"},
		},
	}
	svc := NewConnectionService(nil, stubServerIdentity{id: "server-1"}, store, slog.Default())

	before := time.Now()
	resp, err := svc.ListClusterConnections(context.Background(), "default", "", nil)
	after := time.Now()

	require.NoError(t, err)
	assert.Len(t, resp.Items, 1)

	// The query must exclude records older than the staleness window: notBefore should be
	// roughly now - staleness.
	assert.WithinRange(t, store.listNotBefore,
		before.Add(-DefaultConnectionSnapshotStaleness),
		after.Add(-DefaultConnectionSnapshotStaleness))
}

// TestConnectionService_ListConnectionsPaginationConvention pins the shared pagination
// contract on the node-local path (see model.ListResponse), guarding the previous
// divergence where it emitted a "\xff" end-of-list sentinel token.
func TestConnectionService_ListConnectionsPaginationConvention(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc := NewConnectionService(nil, stubServerIdentity{id: "s1"}, &fakeServerConnectionStore{}, slog.Default())

	const total = 5
	for i := range total {
		conn := agentmodel.NewConnection(fmt.Sprintf("default-conn-%d", i), agentmodel.ConnectionTypeWebSocket)
		conn.SetNamespace("default")
		require.NoError(t, svc.SaveConnection(ctx, conn))
	}
	// Filtered out by namespace.
	other := agentmodel.NewConnection("other-conn", agentmodel.ConnectionTypeWebSocket)
	other.SetNamespace("other")
	require.NoError(t, svc.SaveConnection(ctx, other))

	// Full listing: all items, end-of-list, Continue still a token (not "\xff").
	full, err := svc.ListConnections(ctx, "default", &model.ListOptions{Limit: 0})
	require.NoError(t, err)
	require.Len(t, full.Items, total)
	assert.Equal(t, int64(0), full.RemainingItemCount)
	assert.NotEmpty(t, full.Continue, "Continue is a resume token, non-empty when items are returned")
	assert.NotContains(t, full.Continue, "\xff", "the \\xff end-of-list sentinel must be gone")

	// Empty page: no items, empty Continue.
	empty, err := svc.ListConnections(ctx, "nonexistent", &model.ListOptions{Limit: 10})
	require.NoError(t, err)
	assert.Empty(t, empty.Items)
	assert.Empty(t, empty.Continue)
	assert.Equal(t, int64(0), empty.RemainingItemCount)

	// Paged traversal returns every item exactly once, in the full-listing order.
	pagedIDs := make([]string, 0, total)
	cont := ""

	for pages := 0; ; pages++ {
		require.LessOrEqual(t, pages, total, "pagination must terminate")

		resp, listErr := svc.ListConnections(ctx, "default", &model.ListOptions{Limit: 2, Continue: cont})
		require.NoError(t, listErr)

		for _, item := range resp.Items {
			pagedIDs = append(pagedIDs, item.IDString())
		}

		if resp.RemainingItemCount == 0 {
			break
		}

		cont = resp.Continue
	}

	fullIDs := make([]string, 0, len(full.Items))
	for _, conn := range full.Items {
		fullIDs = append(fullIDs, conn.IDString())
	}

	assert.Equal(t, fullIDs, pagedIDs, "paged traversal must equal the full ordered listing, with no repeats")
}

func TestConnectionService_ListClusterConnectionsPassesServerIDFilter(t *testing.T) {
	t.Parallel()

	store := &fakeServerConnectionStore{}
	svc := NewConnectionService(nil, stubServerIdentity{id: "server-1"}, store, slog.Default())

	_, err := svc.ListClusterConnections(context.Background(), "default", "server-7", nil)
	require.NoError(t, err)

	assert.Equal(t, "server-7", store.listServerID)
}
