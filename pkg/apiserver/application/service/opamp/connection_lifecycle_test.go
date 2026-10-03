//nolint:testpackage // Exercise the interleaving of message processing and close cleanup.
package opamp

import (
	"context"
	"encoding/pem"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opamp-go/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
	"github.com/minuk-dev/opampcommander/pkg/certutil"
)

type lifecycleAgentUsecase struct {
	agentport.AgentUsecase

	mu          sync.Mutex
	agent       *agentmodel.Agent
	liveness    *agentmodel.AgentLiveness
	forgetCalls int
	onGet       func()
}

func (s *lifecycleAgentUsecase) GetAgent(ctx context.Context, uid uuid.UUID) (*agentmodel.Agent, error) {
	if s.onGet != nil {
		s.onGet()
	}

	return s.GetOrCreateAgent(ctx, uid)
}

func (s *lifecycleAgentUsecase) GetOrCreateAgent(context.Context, uuid.UUID) (*agentmodel.Agent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.agent.Clone(), nil
}

func (s *lifecycleAgentUsecase) SaveAgent(_ context.Context, agent *agentmodel.Agent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.agent = agent.Clone()

	return nil
}

func (s *lifecycleAgentUsecase) TouchAgentLiveness(_ context.Context, agent *agentmodel.Agent, _ time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.liveness = agentmodel.NewAgentLivenessFromAgent(agent)

	return false // Heartbeats only touch the fast tier inside the persistence throttle window.
}

func (s *lifecycleAgentUsecase) ForgetAgentLiveness(context.Context, uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.liveness = nil
	s.forgetCalls++

	return nil
}

type lifecycleIdentityProvider struct {
	agentport.ServerIdentityProvider
}

func (*lifecycleIdentityProvider) CurrentServer(context.Context) (*agentmodel.Server, error) {
	return &agentmodel.Server{}, nil
}

type lifecycleBaseConnection interface{ types.Connection }

type lifecycleNetworkConnection struct {
	lifecycleBaseConnection

	raw net.Conn
}

func (c *lifecycleNetworkConnection) Connection() net.Conn { return c.raw }

func lifecycleWire(t *testing.T) *lifecycleNetworkConnection {
	t.Helper()

	raw, peer := net.Pipe()

	t.Cleanup(func() {
		_ = raw.Close()
		_ = peer.Close()
	})

	return &lifecycleNetworkConnection{raw: raw}
}

func TestConnectionCleanupSerializesWithReplacementMessage(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name         string
		cleanupFirst bool
	}{
		{name: "cleanup already running", cleanupFirst: true},
		{name: "cleanup delayed until after replacement", cleanupFirst: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			uid := uuid.New()
			agent := agentmodel.NewAgent(uid)
			agent.Status.Connected = true
			agentUC := &lifecycleAgentUsecase{agent: agent, liveness: agentmodel.NewAgentLivenessFromAgent(agent)}
			connUC := agentservice.NewConnectionService(nil, nil, nil, slog.New(slog.DiscardHandler))
			oldWire, newWire := lifecycleWire(t), lifecycleWire(t)
			old := agentmodel.NewConnection(oldWire, agentmodel.ConnectionTypeWebSocket)
			old.SetInstanceUID(uid)

			newConnection := agentmodel.NewConnection(newWire, agentmodel.ConnectionTypeWebSocket)

			require.NoError(t, connUC.SaveConnection(t.Context(), old))
			require.NoError(t, connUC.SaveConnection(t.Context(), newConnection))
			svc := newTestService(t, agentUC, connUC)
			svc.serverIdentityProvider = &lifecycleIdentityProvider{}
			svc.serverToAgentBuilder = agentservice.NewServerToAgentBuilder(nil, nil, svc.logger)
			message := &protobufs.AgentToServer{InstanceUid: uid[:]}

			if tc.cleanupFirst {
				loadingAgent, releaseCleanup := make(chan struct{}), make(chan struct{})

				var once sync.Once

				agentUC.onGet = func() {
					once.Do(func() {
						close(loadingAgent)
						<-releaseCleanup
					})
				}

				cleanupDone := make(chan error, 1)
				go func() { cleanupDone <- svc.cleanUpConnection(t.Context(), oldWire) }()

				select {
				case <-loadingAgent:
				case <-time.After(time.Second):
					t.Fatal("cleanup did not reach the agent read")
				}

				messageDone := make(chan *protobufs.ServerToAgent, 1)
				go func() { messageDone <- svc.OnMessage(t.Context(), newWire, message) }()
				// Check a real waiter instead of relying on goroutine scheduling or sleeps.
				waiting := assert.Eventually(t, func() bool {
					svc.connectionLifecycleMu.Lock()
					defer svc.connectionLifecycleMu.Unlock()

					lock := svc.connectionLifecycleLocks[uid]

					return lock != nil && lock.users == 2
				}, time.Second, time.Millisecond, "replacement must wait for cleanup's status and liveness writes")

				close(releaseCleanup)
				require.NoError(t, <-cleanupDone)
				require.Nil(t, (<-messageDone).GetErrorResponse())
				require.True(t, waiting)
			} else {
				require.Nil(t, svc.OnMessage(t.Context(), newWire, message).GetErrorResponse())
				require.NoError(t, svc.cleanUpConnection(t.Context(), oldWire))
			}

			active, err := connUC.GetConnectionByInstanceUID(t.Context(), uid)
			require.NoError(t, err)
			assert.Equal(t, newConnection.UID, active.UID)
			_, err = connUC.GetConnectionByID(t.Context(), oldWire)
			require.ErrorIs(t, err, agentport.ErrConnectionNotFound)
			require.NotNil(t, agentUC.liveness)
			assert.True(t, agentUC.liveness.Connected)

			if tc.cleanupFirst {
				assert.Equal(t, 1, agentUC.forgetCalls)
			} else {
				assert.Zero(t, agentUC.forgetCalls, "old close must not remove replacement liveness")
				assert.True(t, agentUC.agent.Status.Connected)
			}

			assert.Empty(t, svc.connectionLifecycleLocks, "idle agent locks must not accumulate")
		})
	}
}

type certificateSaveHookUsecase struct {
	*stubAgentUsecase

	onSave func()
}

func (s *certificateSaveHookUsecase) SaveAgent(ctx context.Context, agent *agentmodel.Agent) error {
	s.onSave()

	return s.stubAgentUsecase.SaveAgent(ctx, agent)
}

func TestClientCertificateReplacementClosesCapturedSocket(t *testing.T) {
	t.Parallel()

	uid := uuid.New()
	agent := agentmodel.NewAgent(uid)
	newDER := []byte("new-leaf")
	agent.Status.ActiveClientCertificateHash = certutil.SHA256Fingerprint([]byte("old-leaf"))
	require.NoError(t, agent.ApplyConnectionSettings(&agentmodel.AgentOpAMPConnectionSettings{
		DestinationEndpoint: "wss://example.test/api/v1/opamp",
		Certificate: &agentmodel.AgentCertificate{
			Cert: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: newDER}),
		},
	}, nil, nil, nil, nil))

	oldClosed, newClosed := false, false
	old := &disconnectTrackingConnection{disconnected: &oldClosed}
	newWire := &disconnectTrackingConnection{disconnected: &newClosed}
	connUC := &stubConnectionUsecase{byInstanceUID: &agentmodel.Connection{
		ID: old, UID: uuid.New(), Type: agentmodel.ConnectionTypeWebSocket, InstanceUID: uid,
	}}
	agentUC := &certificateSaveHookUsecase{
		stubAgentUsecase: &stubAgentUsecase{getResult: agent},
		onSave: func() {
			// Simulate the index changing immediately after the fingerprint commit.
			connUC.byInstanceUID = &agentmodel.Connection{
				ID: newWire, UID: uuid.New(), Type: agentmodel.ConnectionTypeWebSocket, InstanceUID: uid,
			}
		},
	}
	svc := newTestService(t, agentUC, connUC)
	require.True(t, svc.AuthorizeClientCertificate(t.Context(), uid, newDER))
	assert.True(t, oldClosed)
	assert.False(t, newClosed, "the socket using the new certificate must remain open")
}
