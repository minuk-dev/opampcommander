package agentservice_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
)

// newServerServiceForSend builds a ServerService wired with the given persistence, sender
// and identity mocks, and a fixed clock at now. The remaining collaborators are unused by
// the send/dispatch paths these tests exercise.
func newServerServiceForSend(
	mockPersistence *MockServerPersistencePort,
	mockEventSender *MockServerEventSenderPort,
	mockIdentity *MockServerIdentityProvider,
	mockConnection *MockConnectionUsecase,
	mockAgent *MockAgentUsecase,
	now time.Time,
) *agentservice.ServerService {
	svc := agentservice.NewServerService(
		slog.Default(),
		mockPersistence,
		mockEventSender,
		new(MockServerEventReceiverPort),
		mockIdentity,
		mockConnection,
		mockAgent,
		noopAgentCacheInvalidator{},
		agentservice.NewServerToAgentBuilder(nil, nil, slog.Default()),
	)
	svc.SetClock(newTestFakeClock(now))

	return svc
}

func TestServerService_Run_DelegatesToReceiver(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when the receiver stops cleanly", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockReceiver := new(MockServerEventReceiverPort)
		mockReceiver.On("StartReceiver", ctx, mock.Anything).Return(nil)

		svc := agentservice.NewServerService(
			slog.Default(),
			new(MockServerPersistencePort),
			new(MockServerEventSenderPort),
			mockReceiver,
			new(MockServerIdentityProvider),
			new(MockConnectionUsecase),
			new(MockAgentUsecase),
			noopAgentCacheInvalidator{},
			agentservice.NewServerToAgentBuilder(nil, nil, slog.Default()),
		)

		require.NoError(t, svc.Run(ctx))
		mockReceiver.AssertCalled(t, "StartReceiver", ctx, mock.Anything)
	})

	t.Run("swallows a receiver error after logging it", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		mockReceiver := new(MockServerEventReceiverPort)
		mockReceiver.On("StartReceiver", ctx, mock.Anything).Return(errDatabaseError)

		svc := agentservice.NewServerService(
			slog.Default(),
			new(MockServerPersistencePort),
			new(MockServerEventSenderPort),
			mockReceiver,
			new(MockServerIdentityProvider),
			new(MockConnectionUsecase),
			new(MockAgentUsecase),
			noopAgentCacheInvalidator{},
			agentservice.NewServerToAgentBuilder(nil, nil, slog.Default()),
		)

		// Run never surfaces the receiver error to the executor; it logs and returns nil.
		require.NoError(t, svc.Run(ctx))
	})
}

func TestServerService_Name(t *testing.T) {
	t.Parallel()

	svc := newServerServiceForSend(
		new(MockServerPersistencePort),
		new(MockServerEventSenderPort),
		new(MockServerIdentityProvider),
		new(MockConnectionUsecase),
		new(MockAgentUsecase),
		time.Now(),
	)

	assert.Equal(t, "ServerService", svc.Name())
}

func TestServerService_Shutdown_ClearsCache(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	now := time.Now()
	serverID := testServerID

	mockPersistence := new(MockServerPersistencePort)
	mockPersistence.On("GetServer", ctx, serverID).
		Return(&agentmodel.Server{ID: serverID, LastHeartbeatAt: now}, nil).Once()

	svc := newServerServiceForSend(
		mockPersistence,
		new(MockServerEventSenderPort),
		new(MockServerIdentityProvider),
		new(MockConnectionUsecase),
		new(MockAgentUsecase),
		now,
	)

	// Populate the cache, then drop it. A subsequent get must hit persistence again.
	_, err := svc.GetServer(ctx, serverID)
	require.NoError(t, err)

	svc.Shutdown()

	mockPersistence.On("GetServer", ctx, serverID).
		Return(&agentmodel.Server{ID: serverID, LastHeartbeatAt: now}, nil).Once()

	_, err = svc.GetServer(ctx, serverID)
	require.NoError(t, err)

	mockPersistence.AssertNumberOfCalls(t, "GetServer", 2)
}

func TestServerService_ListServers_FiltersDeadServers(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	now := time.Now()

	mockPersistence := new(MockServerPersistencePort)
	mockPersistence.On("ListServers", ctx).Return([]*agentmodel.Server{
		{ID: "alive", LastHeartbeatAt: now},
		{ID: "dead", LastHeartbeatAt: now.Add(-10 * time.Minute)},
	}, nil)

	svc := newServerServiceForSend(
		mockPersistence,
		new(MockServerEventSenderPort),
		new(MockServerIdentityProvider),
		new(MockConnectionUsecase),
		new(MockAgentUsecase),
		now,
	)

	servers, err := svc.ListServers(ctx)
	require.NoError(t, err)
	require.Len(t, servers, 1)
	assert.Equal(t, "alive", servers[0].ID)
}

func TestServerService_ListServers_PropagatesError(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	mockPersistence := new(MockServerPersistencePort)
	mockPersistence.On("ListServers", ctx).Return(nil, errDatabaseError)

	svc := newServerServiceForSend(
		mockPersistence,
		new(MockServerEventSenderPort),
		new(MockServerIdentityProvider),
		new(MockConnectionUsecase),
		new(MockAgentUsecase),
		time.Now(),
	)

	_, err := svc.ListServers(ctx)
	require.ErrorIs(t, err, errDatabaseError)
}

func TestServerService_SendMessageToServerByServerID_ResolvesAndSends(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	now := time.Now()
	remoteServerID := "server-2"

	mockPersistence := new(MockServerPersistencePort)
	mockEventSender := new(MockServerEventSenderPort)
	mockIdentity := new(MockServerIdentityProvider)

	mockIdentity.On("CurrentServerID").Return(testServerID)
	mockPersistence.On("GetServer", ctx, remoteServerID).
		Return(&agentmodel.Server{ID: remoteServerID, LastHeartbeatAt: now}, nil)
	mockEventSender.On("SendMessageToServer", ctx, matchServerID(remoteServerID), mock.Anything).Return(nil)

	svc := newServerServiceForSend(
		mockPersistence,
		mockEventSender,
		mockIdentity,
		new(MockConnectionUsecase),
		new(MockAgentUsecase),
		now,
	)

	msg := serverevent.Message{
		Source: testServerID,
		Target: remoteServerID,
		Type:   serverevent.MessageTypeInvalidateAgentCache,
		Payload: serverevent.MessagePayload{
			MessageForInvalidateAgentCache: &serverevent.MessageForInvalidateAgentCache{
				AgentInstanceUIDs: []uuid.UUID{uuid.New()},
			},
		},
	}

	err := svc.SendMessageToServerByServerID(ctx, remoteServerID, msg)
	require.NoError(t, err)

	mockEventSender.AssertCalled(t, "SendMessageToServer", ctx, matchServerID(remoteServerID), mock.Anything)
}

func TestServerService_SendMessageToServerByServerID_LookupError(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	mockPersistence := new(MockServerPersistencePort)
	mockPersistence.On("GetServer", ctx, "missing").Return(nil, errDatabaseError)

	svc := newServerServiceForSend(
		mockPersistence,
		new(MockServerEventSenderPort),
		new(MockServerIdentityProvider),
		new(MockConnectionUsecase),
		new(MockAgentUsecase),
		time.Now(),
	)

	err := svc.SendMessageToServerByServerID(ctx, "missing", serverevent.Message{})
	require.ErrorIs(t, err, errDatabaseError)
}

func TestServerService_SendMessageToServer_NotAlive(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	now := time.Now()

	svc := newServerServiceForSend(
		new(MockServerPersistencePort),
		new(MockServerEventSenderPort),
		new(MockServerIdentityProvider),
		new(MockConnectionUsecase),
		new(MockAgentUsecase),
		now,
	)

	// A server whose last heartbeat is well past the timeout cannot receive messages.
	dead := &agentmodel.Server{ID: "server-2", LastHeartbeatAt: now.Add(-10 * time.Minute)}

	err := svc.SendMessageToServer(ctx, dead, serverevent.Message{})
	require.ErrorIs(t, err, agentservice.ErrServerNotAlive)
}

func TestServerService_SendMessageToServer_UnknownEventTypeIsIgnoredLocally(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	now := time.Now()

	mockIdentity := new(MockServerIdentityProvider)
	mockIdentity.On("CurrentServerID").Return(testServerID)

	svc := newServerServiceForSend(
		new(MockServerPersistencePort),
		new(MockServerEventSenderPort),
		mockIdentity,
		new(MockConnectionUsecase),
		new(MockAgentUsecase),
		now,
	)

	// Target the current server so the message is dispatched locally, reaching
	// handleServerEvent's default branch for an unrecognized type (no error, no side effect).
	self := &agentmodel.Server{ID: testServerID, LastHeartbeatAt: now}
	msg := serverevent.Message{
		Source: testServerID,
		Target: testServerID,
		Type:   serverevent.MessageType("unknown-type"),
	}

	err := svc.SendMessageToServer(ctx, self, msg)
	require.NoError(t, err)
}

func TestServerService_SendMessageToServer_LocalSendServerToAgent_NilPayload(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	now := time.Now()

	mockIdentity := new(MockServerIdentityProvider)
	mockIdentity.On("CurrentServerID").Return(testServerID)

	svc := newServerServiceForSend(
		new(MockServerPersistencePort),
		new(MockServerEventSenderPort),
		mockIdentity,
		new(MockConnectionUsecase),
		new(MockAgentUsecase),
		now,
	)

	self := &agentmodel.Server{ID: testServerID, LastHeartbeatAt: now}
	// SendServerToAgent event whose payload is missing must surface ErrEventPayloadNil.
	msg := serverevent.Message{
		Source: testServerID,
		Target: testServerID,
		Type:   serverevent.MessageTypeSendServerToAgent,
	}

	err := svc.SendMessageToServer(ctx, self, msg)
	require.ErrorIs(t, err, agentservice.ErrEventPayloadNil)
}
