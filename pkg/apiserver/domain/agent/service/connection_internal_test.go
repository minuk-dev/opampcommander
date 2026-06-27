package agentservice

import (
	"context"
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
	_ context.Context, _ string, notBefore time.Time, _ *model.ListOptions,
) (*model.ListResponse[*agentmodel.ServerConnection], error) {
	f.listNotBefore = notBefore

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
	resp, err := svc.ListClusterConnections(context.Background(), "default", nil)
	after := time.Now()

	require.NoError(t, err)
	assert.Len(t, resp.Items, 1)

	// The query must exclude records older than the staleness window: notBefore should be
	// roughly now - staleness.
	assert.WithinRange(t, store.listNotBefore,
		before.Add(-DefaultConnectionSnapshotStaleness),
		after.Add(-DefaultConnectionSnapshotStaleness))
}
