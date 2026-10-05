//nolint:testpackage // Exercise the interleaving of message processing and close cleanup.
package opamp

import (
	"context"
	"encoding/pem"
	"log/slog"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opamp-go/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	connectionstore "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/store/inmemory"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port/usecasemock"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
	"github.com/minuk-dev/opampcommander/pkg/certutil"
)

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
			agentUC := usecasemock.NewMockAgentUsecase(t)
			store := connectionstore.NewConnectionStore()
			connUC := agentservice.NewConnectionService(nil, store, nil, nil, slog.New(slog.DiscardHandler))
			oldWire, newWire := lifecycleWire(t), lifecycleWire(t)
			old := agentmodel.NewConnection(oldWire, agentmodel.ConnectionTypeWebSocket)
			old.SetInstanceUID(uid)

			newConnection := agentmodel.NewConnection(newWire, agentmodel.ConnectionTypeWebSocket)

			require.NoError(t, connUC.SaveConnection(t.Context(), old))
			require.NoError(t, connUC.SaveConnection(t.Context(), newConnection))
			svc := newTestService(t, agentUC, connUC)
			svc.connectionStore = store
			svc.serverIdentityProvider = &lifecycleIdentityProvider{}
			svc.serverToAgentBuilder = agentservice.NewServerToAgentBuilder(nil, nil, svc.logger)
			// Separate stateless Services share one Store-owned session scope.
			replacementService := newTestService(t, agentUC, connUC)
			replacementService.connectionStore = store
			replacementService.serverIdentityProvider = svc.serverIdentityProvider
			replacementService.serverToAgentBuilder = svc.serverToAgentBuilder
			message := &protobufs.AgentToServer{InstanceUid: uid[:]}

			loadingAgent, releaseCleanup := make(chan struct{}), make(chan struct{})
			messageEntered := make(chan struct{}, 1)

			var paused atomic.Bool

			read := agentUC.EXPECT().GetAgent(mock.Anything, uid).
				RunAndReturn(func(context.Context, uuid.UUID) (*agentmodel.Agent, error) {
					if tc.cleanupFirst && paused.CompareAndSwap(false, true) {
						close(loadingAgent)
						<-releaseCleanup
					}

					return agent.Clone(), nil
				})
			agentUC.EXPECT().GetOrCreateAgent(mock.Anything, uid).
				RunAndReturn(func(context.Context, uuid.UUID) (*agentmodel.Agent, error) {
					messageEntered <- struct{}{}

					return agent.Clone(), nil
				}).Once()
			agentUC.EXPECT().TouchAgentLiveness(mock.Anything, mock.MatchedBy(func(observed *agentmodel.Agent) bool {
				return observed.Metadata.InstanceUID == uid && observed.Status.Connected &&
					observed.Status.ConnectionType == agentmodel.ConnectionTypeWebSocket
			}), mock.Anything).Return(false).Once()

			if tc.cleanupFirst {
				read.Times(2) // Cleanup and the replacement's conflict check each load the agent.
				agentUC.EXPECT().SaveAgent(mock.Anything, mock.MatchedBy(func(saved *agentmodel.Agent) bool {
					return saved.Metadata.InstanceUID == uid && !saved.Status.Connected
				})).Return(nil).Once()
				agentUC.EXPECT().ForgetAgentLiveness(mock.Anything, uid).Return(nil).Once()

				cleanupDone := make(chan error, 1)
				go func() { cleanupDone <- svc.cleanUpConnection(t.Context(), oldWire) }()

				select {
				case <-loadingAgent:
				case <-time.After(time.Second):
					t.Fatal("cleanup did not reach the agent read")
				}

				messageDone := make(chan *protobufs.ServerToAgent, 1)
				go func() { messageDone <- replacementService.OnMessage(t.Context(), newWire, message) }()

				waiting := assert.Never(t, func() bool {
					return len(messageEntered) != 0
				}, 100*time.Millisecond, time.Millisecond, "replacement must wait for cleanup's status and liveness writes")

				close(releaseCleanup)
				require.NoError(t, <-cleanupDone)
				require.Nil(t, (<-messageDone).GetErrorResponse())
				require.True(t, waiting)
			} else {
				read.Once()
				require.Nil(t, replacementService.OnMessage(t.Context(), newWire, message).GetErrorResponse())
				require.NoError(t, svc.cleanUpConnection(t.Context(), oldWire))
			}

			active, err := connUC.GetConnectionByInstanceUID(t.Context(), uid)
			require.NoError(t, err)
			assert.Equal(t, newConnection.UID, active.UID)
			_, err = connUC.GetConnectionByID(t.Context(), oldWire)
			require.ErrorIs(t, err, agentport.ErrConnectionNotFound)

			if !tc.cleanupFirst {
				agentUC.AssertNotCalled(t, "SaveAgent", mock.Anything, mock.Anything)
				agentUC.AssertNotCalled(t, "ForgetAgentLiveness", mock.Anything, uid)
			}
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
