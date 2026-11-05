//nolint:testpackage // white-box test of unexported instanceUID conflict helpers
package opamp

import (
	"context"
	"crypto/sha256"
	"encoding/pem"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/open-telemetry/opamp-go/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	connectionstore "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/store/inmemory"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/certutil"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

// stubAgentUsecase satisfies agentport.AgentUsecase by embedding the interface; only the
// methods used by the certificate authorization tests are overridden.
type stubAgentUsecase struct {
	agentport.AgentUsecase

	getResult *agentmodel.Agent
	getErr    error
	saved     *agentmodel.Agent
}

func (s *stubAgentUsecase) GetAgent(_ context.Context, _ uuid.UUID) (*agentmodel.Agent, error) {
	return s.getResult, s.getErr
}

func (s *stubAgentUsecase) GetOrCreateAgent(_ context.Context, _ uuid.UUID) (*agentmodel.Agent, error) {
	return s.getResult, s.getErr
}

func (s *stubAgentUsecase) InvalidateCache(uuid.UUID) {}

func (s *stubAgentUsecase) SaveAgent(_ context.Context, a *agentmodel.Agent) error {
	s.saved = a

	return nil
}

// stubConnectionUsecase satisfies agentport.ConnectionUsecase the same way.
type stubConnectionUsecase struct {
	agentport.ConnectionUsecase

	byInstanceUID    *agentmodel.Connection
	byInstanceUIDErr error
	byID             *agentmodel.Connection
	byIDErr          error
}

func (s *stubConnectionUsecase) GetConnectionByInstanceUID(
	_ context.Context,
	_ uuid.UUID,
) (*agentmodel.Connection, error) {
	return s.byInstanceUID, s.byInstanceUIDErr
}

func (s *stubConnectionUsecase) GetConnectionByID(_ context.Context, _ any) (*agentmodel.Connection, error) {
	return s.byID, s.byIDErr
}

func newTestService(t *testing.T, agentUC agentport.AgentUsecase, connUC agentport.ConnectionUsecase) *Service {
	t.Helper()

	return &Service{
		clock:             clock.NewRealClock(),
		logger:            slog.New(slog.DiscardHandler),
		agentUsecase:      agentUC,
		connectionUsecase: connUC,
		connectionStore:   connectionstore.NewConnectionStore(),
	}
}

func TestClientCertificateReplacedOnlyAfterNewConnection(t *testing.T) {
	t.Parallel()

	agent := agentmodel.NewAgent(uuid.New())
	newDER := []byte("new-leaf")
	require.NoError(t, agent.ApplyConnectionSettings(&agentmodel.AgentOpAMPConnectionSettings{
		DestinationEndpoint: "wss://example.test/api/v1/opamp",
		Certificate: &agentmodel.AgentCertificate{
			Cert: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: newDER}),
		},
	}, nil, nil, nil, nil))
	stub := &stubAgentUsecase{getResult: agent}
	svc := newTestService(t, stub, nil)
	uid := agent.Metadata.InstanceUID
	assert.True(t, svc.AuthorizeClientCertificate(t.Context(), uid, []byte("old-leaf")))

	oldHash := sha256.Sum256([]byte("old-leaf"))
	assert.Equal(t, oldHash[:], stub.saved.Status.ActiveClientCertificateHash)

	agent.Status.ConnectionSettingsStatus.Status = agentmodel.ConnectionSettingsStatusApplied
	agent.Status.ConnectionSettingsStatus.LastConnectionSettingsHash = agent.Spec.ConnectionInfo.Hash.Bytes()

	assert.True(t, svc.AuthorizeClientCertificate(t.Context(), uid, []byte("old-leaf")))
	assert.False(t, svc.AuthorizeClientCertificate(t.Context(), uid, []byte("unoffered-leaf")))
	assert.True(t, svc.AuthorizeClientCertificate(t.Context(), uid, newDER))
	assert.False(t, svc.AuthorizeClientCertificate(t.Context(), uid, []byte("old-leaf")))
	// A later offer must not reactivate an older certificate.
	agent.Status.ConnectionSettingsStatus.Status = agentmodel.ConnectionSettingsStatusUnset

	assert.False(t, svc.AuthorizeClientCertificate(t.Context(), uid, []byte("old-leaf")))
}

type disconnectTrackingConnection struct {
	disconnected *bool
}

func (c *disconnectTrackingConnection) Connection() net.Conn { return nil }
func (c *disconnectTrackingConnection) Send(context.Context, *protobufs.ServerToAgent) error {
	return nil
}

func (c *disconnectTrackingConnection) Disconnect() error {
	*c.disconnected = true

	return nil
}

func TestClientCertificateReplacementClosesOldWebSocket(t *testing.T) {
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

	disconnected := false
	old := &disconnectTrackingConnection{disconnected: &disconnected}
	connUC := &stubConnectionUsecase{byInstanceUID: &agentmodel.Connection{
		ID: old, Type: agentmodel.ConnectionTypeWebSocket, InstanceUID: uid,
	}}
	require.Implements(t, (*types.Connection)(nil), old)
	svc := newTestService(t, &stubAgentUsecase{getResult: agent}, connUC)
	assert.True(t, svc.AuthorizeClientCertificate(t.Context(), uid, newDER))
	assert.True(t, disconnected)
}

func TestRevokedAgentCertificateRejected(t *testing.T) {
	t.Parallel()
	svc := newTestService(t, &stubAgentUsecase{getErr: model.ErrAgentRevoked}, nil)
	assert.False(t, svc.AuthorizeClientCertificate(t.Context(), uuid.New(), []byte("old-leaf")))
}

func aliveConn(t *testing.T, uid uuid.UUID, instanceUID uuid.UUID) *agentmodel.Connection {
	t.Helper()

	return &agentmodel.Connection{
		ID:                 struct{}{},
		Type:               agentmodel.ConnectionTypeWebSocket,
		UID:                uid,
		InstanceUID:        instanceUID,
		LastCommunicatedAt: time.Now(),
	}
}

func TestHandleInstanceUIDConflict(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name           string
		transport      agentmodel.ConnectionType
		sameConnection bool
		closed         bool
		missing        bool
		unregistered   bool
		blocked        bool
	}{
		{name: "new agent", missing: true},
		{name: "established WebSocket", transport: agentmodel.ConnectionTypeWebSocket, sameConnection: true},
		{name: "closed WebSocket awaiting cleanup", transport: agentmodel.ConnectionTypeWebSocket, closed: true},
		{name: "HTTP poll", transport: agentmodel.ConnectionTypeHTTP},
		{name: "competing WebSocket", transport: agentmodel.ConnectionTypeWebSocket, blocked: true},
		{
			name: "unregistered competing connection", transport: agentmodel.ConnectionTypeWebSocket,
			unregistered: true, blocked: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			uid := uuid.New()
			existing := aliveConn(t, uuid.New(), uid)
			existing.Type = tc.transport
			existing.Closed = tc.closed

			current := aliveConn(t, uuid.New(), uid)
			if tc.sameConnection {
				current = existing
			}

			if tc.missing {
				existing = nil
			}

			if tc.unregistered {
				current = nil
			}

			// Session admission must not read or mutate the agent's description or desired state.
			svc := newTestService(t, nil, &stubConnectionUsecase{byInstanceUID: existing, byID: current})

			response := svc.handleInstanceUIDConflict(t.Context(), svc.logger, nil, uid)
			if !tc.blocked {
				assert.Nil(t, response)

				return
			}

			require.NotNil(t, response)
			assert.Equal(t, uid[:], response.GetInstanceUid())
			require.NotNil(t, response.GetErrorResponse())
			assert.Equal(t, protobufs.ServerErrorResponseType_ServerErrorResponseType_Unavailable,
				response.GetErrorResponse().GetType())
			assert.Nil(t, response.GetAgentIdentification(), "overlapping reconnects must keep their UID")
		})
	}
}

func TestHandleInstanceUIDConflict_ReconnectBeforeCloseCleanup(t *testing.T) {
	t.Parallel()

	uid := uuid.New()
	store := connectionstore.NewConnectionStore()
	connUC := agentservice.NewConnectionService(nil, store, nil, nil, slog.New(slog.DiscardHandler))
	svc := newTestService(t, nil, connUC)
	svc.connectionStore = store
	oldWire := lifecycleWire(t)
	old := agentmodel.NewConnection(oldWire, agentmodel.ConnectionTypeWebSocket)
	old.SetInstanceUID(uid)
	require.NoError(t, store.Put(t.Context(), old))

	newWire := lifecycleWire(t)
	response := svc.handleInstanceUIDConflict(t.Context(), svc.logger, newWire, uid)
	require.NotNil(t, response)
	assert.Nil(t, response.GetAgentIdentification())
	require.NotNil(t, response.GetErrorResponse())
	assert.Equal(t, protobufs.ServerErrorResponseType_ServerErrorResponseType_Unavailable,
		response.GetErrorResponse().GetType())

	svc.OnConnectionClose(oldWire)
	// Deliberately do not run the asynchronous cleanup worker.
	assert.Nil(t, svc.handleInstanceUIDConflict(t.Context(), svc.logger, newWire, uid))
}
