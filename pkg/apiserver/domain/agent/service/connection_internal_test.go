package agentservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opamp-go/server/types"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// errFakeSend is a static error the fake WebSocket connection returns from Send.
var errFakeSend = errors.New("fake send failure")

// fakeConnection implements opamp types.Connection for the connection-type and
// send tests. netConn nil models an HTTP (request-response) connection; a real
// net.Conn models a persistent WebSocket.
type fakeConnection struct {
	netConn net.Conn
	sendErr error
	sent    []*protobufs.ServerToAgent
}

func (f *fakeConnection) Connection() net.Conn { return f.netConn }

func (f *fakeConnection) Send(_ context.Context, message *protobufs.ServerToAgent) error {
	f.sent = append(f.sent, message)

	return f.sendErr
}

func (f *fakeConnection) Disconnect() error { return nil }

var _ types.Connection = (*fakeConnection)(nil)

func newTestConnectionService() *Service {
	return NewConnectionService(nil, stubServerIdentity{id: "server-1"}, &fakeServerConnectionStore{}, slog.Default())
}

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
	connID := func(conn *agentmodel.Connection, _ int) string { return conn.IDString() }

	var pagedIDs []string

	cont := ""

	for pages := 0; ; pages++ {
		require.LessOrEqual(t, pages, total, "pagination must terminate")

		resp, listErr := svc.ListConnections(ctx, "default", &model.ListOptions{Limit: 2, Continue: cont})
		require.NoError(t, listErr)

		pagedIDs = append(pagedIDs, lo.Map(resp.Items, connID)...)

		if resp.RemainingItemCount == 0 {
			break
		}

		cont = resp.Continue
	}

	assert.Equal(t, lo.Map(full.Items, connID), pagedIDs,
		"paged traversal must equal the full ordered listing, with no repeats")
}

func TestConnectionService_ListClusterConnectionsPassesServerIDFilter(t *testing.T) {
	t.Parallel()

	store := &fakeServerConnectionStore{}
	svc := NewConnectionService(nil, stubServerIdentity{id: "server-1"}, store, slog.Default())

	_, err := svc.ListClusterConnections(context.Background(), "default", "server-7", nil)
	require.NoError(t, err)

	assert.Equal(t, "server-7", store.listServerID)
}

func TestConnectionService_Name(t *testing.T) {
	t.Parallel()

	assert.Equal(t, connectionServiceName, newTestConnectionService().Name())
}

func TestConnectionService_GetConnectionByID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc := newTestConnectionService()

	conn := agentmodel.NewConnection("conn-a", agentmodel.ConnectionTypeWebSocket)
	conn.SetNamespace("default")
	require.NoError(t, svc.SaveConnection(ctx, conn))

	got, err := svc.GetConnectionByID(ctx, "conn-a")
	require.NoError(t, err)
	assert.Equal(t, conn.UID, got.UID)

	_, err = svc.GetConnectionByID(ctx, "missing")
	require.ErrorIs(t, err, agentport.ErrConnectionNotFound)
}

func TestConnectionService_DeleteConnection(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc := newTestConnectionService()

	conn := agentmodel.NewConnection("conn-a", agentmodel.ConnectionTypeWebSocket)
	conn.SetNamespace("default")
	require.NoError(t, svc.SaveConnection(ctx, conn))

	require.NoError(t, svc.DeleteConnection(ctx, conn))

	_, err := svc.GetConnectionByID(ctx, "conn-a")
	require.ErrorIs(t, err, agentport.ErrConnectionNotFound)
}

func TestConnectionService_GetOrCreateConnectionByID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc := newTestConnectionService()

	existing := agentmodel.NewConnection("conn-a", agentmodel.ConnectionTypeWebSocket)
	existing.SetNamespace("default")
	require.NoError(t, svc.SaveConnection(ctx, existing))

	got, err := svc.GetOrCreateConnectionByID(ctx, "conn-a")
	require.NoError(t, err)
	assert.Equal(t, existing.UID, got.UID, "existing connection is returned as-is")

	// A brand-new id yields a fresh connection (not persisted); a plain string is
	// not an opamp connection, so its type is Unknown.
	created, err := svc.GetOrCreateConnectionByID(ctx, "brand-new")
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, agentmodel.ConnectionTypeUnknown, created.Type)
}

func TestConnectionService_GetConnectionByInstanceUID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc := newTestConnectionService()

	instanceUID := uuid.New()
	conn := agentmodel.NewConnection("conn-a", agentmodel.ConnectionTypeWebSocket)
	conn.SetInstanceUID(instanceUID)
	conn.SetNamespace("default")
	require.NoError(t, svc.SaveConnection(ctx, conn))

	got, err := svc.GetConnectionByInstanceUID(ctx, instanceUID)
	require.NoError(t, err)
	assert.Equal(t, conn.UID, got.UID)

	_, err = svc.GetConnectionByInstanceUID(ctx, uuid.New())
	require.ErrorIs(t, err, agentport.ErrConnectionNotFound)
}

func TestConnectionService_detectConnectionType(t *testing.T) {
	t.Parallel()

	svc := newTestConnectionService()

	assert.Equal(t, agentmodel.ConnectionTypeUnknown, svc.detectConnectionType("plain-string"),
		"a non-opamp id has an unknown type")
	assert.Equal(t, agentmodel.ConnectionTypeHTTP, svc.detectConnectionType(&fakeConnection{netConn: nil}),
		"no underlying net.Conn means HTTP")

	client, server := net.Pipe()

	t.Cleanup(func() { _ = client.Close(); _ = server.Close() })

	assert.Equal(t, agentmodel.ConnectionTypeWebSocket, svc.detectConnectionType(&fakeConnection{netConn: client}),
		"a persistent net.Conn means WebSocket")
}

func TestConnectionService_SendServerToAgent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("sends over the websocket connection", func(t *testing.T) {
		t.Parallel()

		svc := newTestConnectionService()
		instanceUID := uuid.New()
		fake := &fakeConnection{netConn: nil}
		conn := agentmodel.NewConnection(fake, agentmodel.ConnectionTypeWebSocket)
		conn.SetInstanceUID(instanceUID)
		conn.SetNamespace("default")
		require.NoError(t, svc.SaveConnection(ctx, conn))

		require.NoError(t, svc.SendServerToAgent(ctx, instanceUID, &protobufs.ServerToAgent{}))
		assert.Len(t, fake.sent, 1)
	})

	t.Run("propagates the send error", func(t *testing.T) {
		t.Parallel()

		svc := newTestConnectionService()
		instanceUID := uuid.New()
		fake := &fakeConnection{netConn: nil, sendErr: errFakeSend}
		conn := agentmodel.NewConnection(fake, agentmodel.ConnectionTypeWebSocket)
		conn.SetInstanceUID(instanceUID)
		conn.SetNamespace("default")
		require.NoError(t, svc.SaveConnection(ctx, conn))

		require.ErrorIs(t, svc.SendServerToAgent(ctx, instanceUID, &protobufs.ServerToAgent{}), errFakeSend)
	})

	t.Run("errors when the connection id is not an opamp connection", func(t *testing.T) {
		t.Parallel()

		svc := newTestConnectionService()
		instanceUID := uuid.New()
		conn := agentmodel.NewConnection("string-id", agentmodel.ConnectionTypeWebSocket)
		conn.SetInstanceUID(instanceUID)
		conn.SetNamespace("default")
		require.NoError(t, svc.SaveConnection(ctx, conn))

		var notFound *ConnectionNotFoundError
		require.ErrorAs(t, svc.SendServerToAgent(ctx, instanceUID, &protobufs.ServerToAgent{}), &notFound)
	})

	t.Run("errors when the agent has no connection", func(t *testing.T) {
		t.Parallel()

		svc := newTestConnectionService()
		require.Error(t, svc.SendServerToAgent(ctx, uuid.New(), &protobufs.ServerToAgent{}))
	})
}

func TestConnectionService_effectiveSnapshotSettings(t *testing.T) {
	t.Parallel()

	svc := newTestConnectionService()
	assert.Equal(t, DefaultConnectionSnapshotInterval, svc.effectiveSnapshotInterval())
	assert.Equal(t, DefaultConnectionSnapshotStaleness, svc.effectiveSnapshotStaleness())

	// Non-positive values fall back to the defaults.
	svc.snapshotInterval = 0
	svc.snapshotStaleness = -1
	assert.Equal(t, DefaultConnectionSnapshotInterval, svc.effectiveSnapshotInterval())
	assert.Equal(t, DefaultConnectionSnapshotStaleness, svc.effectiveSnapshotStaleness())

	// Positive values are used verbatim.
	svc.snapshotInterval = 7 * time.Second
	assert.Equal(t, 7*time.Second, svc.effectiveSnapshotInterval())
}

func TestConnectionService_RunClearsSnapshotOnCancel(t *testing.T) {
	t.Parallel()

	store := &fakeServerConnectionStore{}
	svc := NewConnectionService(nil, stubServerIdentity{id: "server-1"}, store, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Run must observe the cancelled ctx and clear this server's snapshot.

	require.NoError(t, svc.Run(ctx))
	assert.Equal(t, "server-1", store.replacedServerID)
	assert.Nil(t, store.replaced, "shutdown clears the server's records")
}

func TestConnectionService_clearSnapshotOnShutdownSkipsWithoutIdentity(t *testing.T) {
	t.Parallel()

	store := &fakeServerConnectionStore{}
	svc := NewConnectionService(nil, stubServerIdentity{id: ""}, store, slog.Default())

	svc.clearSnapshotOnShutdown(context.Background())
	assert.Empty(t, store.replacedServerID, "no identity means no clear call")
}

func TestConnectionErrors(t *testing.T) {
	t.Parallel()

	notSupported := &NotSupportedConnectionTypeError{ConnectionType: agentmodel.ConnectionTypeHTTP}
	assert.Contains(t, notSupported.Error(), "not supported")

	instanceUID := uuid.New()
	notFound := &ConnectionNotFoundError{InstanceUID: instanceUID}
	assert.Contains(t, notFound.Error(), instanceUID.String())
}

// erroringServerConnectionStore fails every persistence call, exercising the
// error branches of the snapshot/cluster paths.
type erroringServerConnectionStore struct{ err error }

func (e *erroringServerConnectionStore) ReplaceServerConnections(
	_ context.Context, _ string, _ []*agentmodel.ServerConnection,
) error {
	return e.err
}

func (e *erroringServerConnectionStore) ListServerConnections(
	_ context.Context, _ string, _ string, _ time.Time, _ *model.ListOptions,
) (*model.ListResponse[*agentmodel.ServerConnection], error) {
	return nil, e.err
}

func TestConnectionService_PersistenceErrorsAreHandled(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := &erroringServerConnectionStore{err: errFakeSend}
	svc := NewConnectionService(nil, stubServerIdentity{id: "server-1"}, store, slog.Default())

	// ListClusterConnections surfaces the persistence error.
	_, err := svc.ListClusterConnections(ctx, "default", "", nil)
	require.Error(t, err)

	// snapshotConnections and clearSnapshotOnShutdown swallow the error (best-effort).
	assert.NotPanics(t, func() { svc.snapshotConnections(ctx) })
	assert.NotPanics(t, func() { svc.clearSnapshotOnShutdown(ctx) })
}

func TestConnectionService_ListConnectionsNilOptions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc := newTestConnectionService()

	conn := agentmodel.NewConnection("conn-a", agentmodel.ConnectionTypeWebSocket)
	conn.SetNamespace("default")
	require.NoError(t, svc.SaveConnection(ctx, conn))

	resp, err := svc.ListConnections(ctx, "default", nil)
	require.NoError(t, err)
	assert.Len(t, resp.Items, 1)
}

// signalingStore reports each ReplaceServerConnections call on a channel so a test
// can observe the periodic snapshot tick deterministically.
type signalingStore struct{ replaced chan struct{} }

func (s *signalingStore) ReplaceServerConnections(
	_ context.Context, _ string, _ []*agentmodel.ServerConnection,
) error {
	select {
	case s.replaced <- struct{}{}:
	default:
	}

	return nil
}

func (s *signalingStore) ListServerConnections(
	_ context.Context, _ string, _ string, _ time.Time, _ *model.ListOptions,
) (*model.ListResponse[*agentmodel.ServerConnection], error) {
	//exhaustruct:ignore
	return &model.ListResponse[*agentmodel.ServerConnection]{}, nil
}

func TestConnectionService_RunSnapshotsOnTick(t *testing.T) {
	t.Parallel()

	store := &signalingStore{replaced: make(chan struct{}, 1)}
	svc := NewConnectionService(nil, stubServerIdentity{id: "server-1"}, store, slog.Default())
	svc.snapshotInterval = time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)

	go func() { done <- svc.Run(ctx) }()

	select {
	case <-store.replaced:
	case <-time.After(2 * time.Second):
		t.Fatal("expected a snapshot within the interval")
	}

	cancel()
	require.NoError(t, <-done)
}
