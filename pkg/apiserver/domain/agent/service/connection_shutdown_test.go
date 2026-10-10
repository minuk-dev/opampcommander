package agentservice_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	connectionstore "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/store/inmemory"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
)

var (
	errCloseConnection = errors.New("close failure")
	errListConnections = errors.New("list failure")
)

type shutdownTransport struct {
	mu       sync.Mutex
	admitted func()
	closed   map[any]time.Time
	closeErr error
}

func (s *shutdownTransport) StopAccepting(context.Context) error {
	if s.admitted != nil {
		s.admitted()
	}

	return nil
}
func (s *shutdownTransport) CloseConnection(_ context.Context, id any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.closed[id] = time.Now()

	return s.closeErr
}

func TestConnectionService_CloseLocalConnections(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		store := connectionstore.NewConnectionStore()
		require.NoError(t, store.Put(t.Context(), agentmodel.NewConnection("http", agentmodel.ConnectionTypeHTTP)))
		require.NoError(t, store.Put(t.Context(), agentmodel.NewConnection("anonymous", agentmodel.ConnectionTypeWebSocket)))

		transport := &shutdownTransport{closed: make(map[any]time.Time)}
		transport.admitted = func() {
			// An upgrade accepted before shutdown must be registered before the snapshot.
			require.NoError(t, store.Put(t.Context(), agentmodel.NewConnection("late", agentmodel.ConnectionTypeWebSocket)))
		}
		service := agentservice.NewConnectionService(nil, store, nil, nil, slog.Default(), transport)
		start := time.Now()

		done := make(chan error, 1)
		go func() { done <- service.CloseLocalConnections(t.Context(), time.Second) }()

		synctest.Wait()
		transport.mu.Lock()
		assert.Empty(t, transport.closed, "connections must be staggered")
		transport.mu.Unlock()
		time.Sleep(time.Second)
		require.NoError(t, <-done)
		require.Len(t, transport.closed, 2)
		assert.NotContains(t, transport.closed, "http")

		for _, id := range []string{"anonymous", "late"} {
			closedAt, ok := transport.closed[id]
			require.True(t, ok)
			assert.True(t, closedAt.After(start))
			assert.False(t, closedAt.After(start.Add(time.Second)))
		}
	})
}

func TestConnectionService_ClosesAllAfterDeadlineAndErrors(t *testing.T) {
	t.Parallel()

	store := connectionstore.NewConnectionStore()
	require.NoError(t, store.Put(t.Context(), agentmodel.NewConnection("first", agentmodel.ConnectionTypeWebSocket)))
	require.NoError(t, store.Put(t.Context(), agentmodel.NewConnection("second", agentmodel.ConnectionTypeWebSocket)))

	transport := &shutdownTransport{closed: make(map[any]time.Time), closeErr: errCloseConnection}
	service := agentservice.NewConnectionService(nil, store, nil, nil, slog.Default(), transport)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, service.CloseLocalConnections(ctx, time.Hour), errCloseConnection)
	assert.Len(t, transport.closed, 2, "a close failure or deadline must not skip the remaining connections")
}

type failedConnectionStore struct {
	agentport.ConnectionStore

	err error
}

func (s *failedConnectionStore) ListConnections(context.Context) ([]*agentmodel.Connection, error) {
	return nil, s.err
}

func TestConnectionService_ListFailure(t *testing.T) {
	t.Parallel()

	stopped := false
	transport := &shutdownTransport{admitted: func() { stopped = true }}
	service := agentservice.NewConnectionService(
		nil, &failedConnectionStore{err: errListConnections}, nil, nil, slog.Default(), transport,
	)
	require.ErrorIs(t, service.CloseLocalConnections(t.Context(), 0), errListConnections)
	assert.True(t, stopped, "admission must stop before listing local connections")
}
