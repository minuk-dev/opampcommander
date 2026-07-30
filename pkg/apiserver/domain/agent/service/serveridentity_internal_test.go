package agentservice

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var errServerStore = errors.New("server store failure")

// identityFakeClock is a fixed clock for deterministic heartbeat/liveness assertions.
type identityFakeClock struct {
	now time.Time
}

func (c *identityFakeClock) Now() time.Time                       { return c.now }
func (c *identityFakeClock) Since(t time.Time) time.Duration      { return c.now.Sub(t) }
func (c *identityFakeClock) After(time.Duration) <-chan time.Time { return nil }
func (c *identityFakeClock) NewTimer(time.Duration) clock.Timer   { return nil }
func (c *identityFakeClock) Sleep(time.Duration)                  {}
func (c *identityFakeClock) Tick(time.Duration) <-chan time.Time  { return nil }

// fakeServerStore is a stateful in-memory ServerPersistencePort. It lets the identity
// service's register→heartbeat flow be driven end to end without a database, and can be
// told to fail specific operations to exercise the error paths.
type fakeServerStore struct {
	mu       sync.Mutex
	server   *agentmodel.Server
	getErr   error
	putErr   error
	putCount int
	putCh    chan struct{}
}

func (s *fakeServerStore) GetServer(_ context.Context, _ string) (*agentmodel.Server, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.getErr != nil {
		return nil, s.getErr
	}

	if s.server == nil {
		return nil, model.ErrResourceNotExist
	}

	return s.server.Clone(), nil
}

func (s *fakeServerStore) PutServer(_ context.Context, server *agentmodel.Server) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.putErr != nil {
		return s.putErr
	}

	s.server = server.Clone()
	s.putCount++

	if s.putCh != nil {
		select {
		case s.putCh <- struct{}{}:
		default:
		}
	}

	return nil
}

func (s *fakeServerStore) ListServers(_ context.Context) ([]*agentmodel.Server, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.server == nil {
		return nil, nil
	}

	return []*agentmodel.Server{s.server.Clone()}, nil
}

func newIdentityService(store *fakeServerStore, now time.Time) *ServerIdentityService {
	svc := NewServerIdentityService(
		store,
		agentmodel.ServerID("server-1"),
		agentmodel.ServerAddress("10.0.0.5:8081"),
		slog.New(slog.DiscardHandler),
	)
	svc.clock = &identityFakeClock{now: now}

	return svc
}

func TestServerIdentityService_CurrentServerIDAndName(t *testing.T) {
	t.Parallel()

	svc := newIdentityService(&fakeServerStore{}, time.Now())

	assert.Equal(t, "server-1", svc.CurrentServerID())
	assert.Equal(t, "ServerIdentityService", svc.Name())
}

func TestServerIdentityService_CurrentServer(t *testing.T) {
	t.Parallel()

	now := time.Now()

	t.Run("returns the persisted server", func(t *testing.T) {
		t.Parallel()

		store := &fakeServerStore{server: &agentmodel.Server{ID: "server-1", LastHeartbeatAt: now}}
		svc := newIdentityService(store, now)

		server, err := svc.CurrentServer(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "server-1", server.ID)
	})

	t.Run("propagates a persistence error", func(t *testing.T) {
		t.Parallel()

		store := &fakeServerStore{getErr: errServerStore}
		svc := newIdentityService(store, now)

		_, err := svc.CurrentServer(t.Context())
		require.ErrorIs(t, err, errServerStore)
	})
}

func TestServerIdentityService_RegisterServer(t *testing.T) {
	t.Parallel()

	now := time.Now()

	t.Run("registers a brand-new server as alive", func(t *testing.T) {
		t.Parallel()

		store := &fakeServerStore{}
		svc := newIdentityService(store, now)

		require.NoError(t, svc.registerServer(t.Context()))

		require.NotNil(t, store.server)
		assert.Equal(t, "server-1", store.server.ID)
		assert.Equal(t, "10.0.0.5:8081", store.server.Address)
		assert.Equal(t, now, store.server.LastHeartbeatAt)
		assert.True(t, store.server.IsConditionTrue(model.ConditionTypeAlive))
		// New servers are marked as created/registered.
		assert.NotNil(t, store.server.GetCondition(model.ConditionTypeCreated))
	})

	t.Run("re-registers over a dead server, preserving its conditions", func(t *testing.T) {
		t.Parallel()

		dead := &agentmodel.Server{
			ID:              "server-1",
			LastHeartbeatAt: now.Add(-10 * time.Minute),
		}
		dead.MarkRegistered("original-actor")
		store := &fakeServerStore{server: dead}
		svc := newIdentityService(store, now)

		require.NoError(t, svc.registerServer(t.Context()))

		require.NotNil(t, store.server)
		assert.Equal(t, now, store.server.LastHeartbeatAt)
		assert.True(t, store.server.IsConditionTrue(model.ConditionTypeAlive))
		// The original registration actor is preserved rather than overwritten.
		assert.Equal(t, "original-actor", store.server.GetRegisteredBy())
	})

	t.Run("refuses to steal an ID held by an alive server", func(t *testing.T) {
		t.Parallel()

		alive := &agentmodel.Server{ID: "server-1", LastHeartbeatAt: now}
		store := &fakeServerStore{server: alive}
		svc := newIdentityService(store, now)

		err := svc.registerServer(t.Context())
		require.ErrorIs(t, err, ErrServerIDAlreadyExists)
	})

	t.Run("propagates a non-not-found lookup error", func(t *testing.T) {
		t.Parallel()

		store := &fakeServerStore{getErr: errServerStore}
		svc := newIdentityService(store, now)

		require.ErrorIs(t, svc.registerServer(t.Context()), errServerStore)
	})

	t.Run("propagates a persistence write error", func(t *testing.T) {
		t.Parallel()

		store := &fakeServerStore{putErr: errServerStore}
		svc := newIdentityService(store, now)

		require.ErrorIs(t, svc.registerServer(t.Context()), errServerStore)
	})
}

func TestServerIdentityService_SendHeartbeat(t *testing.T) {
	t.Parallel()

	now := time.Now()

	t.Run("refreshes heartbeat time and advertised address", func(t *testing.T) {
		t.Parallel()

		store := &fakeServerStore{server: &agentmodel.Server{
			ID:              "server-1",
			Address:         "", // simulate an older record with no advertised address
			LastHeartbeatAt: now.Add(-time.Minute),
		}}
		svc := newIdentityService(store, now)

		require.NoError(t, svc.sendHeartbeat(t.Context()))

		assert.Equal(t, now, store.server.LastHeartbeatAt)
		assert.Equal(t, "10.0.0.5:8081", store.server.Address)
	})

	t.Run("propagates a lookup error", func(t *testing.T) {
		t.Parallel()

		store := &fakeServerStore{getErr: errServerStore}
		svc := newIdentityService(store, now)

		require.ErrorIs(t, svc.sendHeartbeat(t.Context()), errServerStore)
	})

	t.Run("propagates a write error", func(t *testing.T) {
		t.Parallel()

		store := &fakeServerStore{
			server: &agentmodel.Server{ID: "server-1", LastHeartbeatAt: now},
			putErr: errServerStore,
		}
		svc := newIdentityService(store, now)

		require.ErrorIs(t, svc.sendHeartbeat(t.Context()), errServerStore)
	})
}

func TestServerIdentityService_Run(t *testing.T) {
	t.Parallel()

	now := time.Now()

	t.Run("registers then sends heartbeats until the context is cancelled", func(t *testing.T) {
		t.Parallel()

		store := &fakeServerStore{putCh: make(chan struct{}, 1)}
		svc := newIdentityService(store, now)
		svc.heartbeatInterval = 5 * time.Millisecond

		ctx, cancel := context.WithCancel(t.Context())

		done := make(chan error, 1)

		go func() { done <- svc.Run(ctx) }()

		// First put is the registration; wait for a later put from the heartbeat loop.
		<-store.putCh
		<-store.putCh

		cancel()
		require.NoError(t, <-done)

		store.mu.Lock()
		putCount := store.putCount
		store.mu.Unlock()
		assert.GreaterOrEqual(t, putCount, 2)
	})

	t.Run("fails fast when registration fails", func(t *testing.T) {
		t.Parallel()

		alive := &agentmodel.Server{ID: "server-1", LastHeartbeatAt: now}
		store := &fakeServerStore{server: alive}
		svc := newIdentityService(store, now)

		err := svc.Run(t.Context())
		require.ErrorIs(t, err, ErrServerIDAlreadyExists)
	})
}
