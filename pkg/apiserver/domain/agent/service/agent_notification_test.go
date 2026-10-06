package agentservice_test

import (
	"context"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/store/inmemory"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
)

func TestAgentNotificationService_SharedStoreAndRestart(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		now := time.Now()
		persistence := new(MockServerPersistencePort)
		persistence.On("GetServer", mock.Anything, "target").Return(&agentmodel.Server{
			ID: "target", LastHeartbeatAt: now,
		}, nil)

		identity := new(MockServerIdentityProvider)
		identity.On("CurrentServerID").Return(testServerID)

		sender := new(MockServerEventSenderPort)
		sent := make(chan serverevent.Message, 2)

		sender.On("SendMessageToServer", mock.Anything, mock.Anything, mock.Anything).
			Run(func(args mock.Arguments) {
				message, ok := args.Get(2).(serverevent.Message)
				assert.True(t, ok)

				sent <- message
			}).Return(nil).Twice()
		server := newServerServiceForSend(persistence, sender, identity,
			new(MockConnectionUsecase), new(MockAgentUsecase), now)
		store := inmemory.NewNotificationStore(agentservice.DefaultNotificationDispatchQueue)
		producer := agentservice.NewAgentNotificationService(server, server, identity, store, slog.Default())
		consumer := agentservice.NewAgentNotificationService(server, server, identity, store, slog.Default())

		// One service instance enqueues while another consumes the shared Store.
		// Reusing the consumer after cancellation must not reuse a closed channel or
		// a WaitGroup owned by its previous Run invocation.
		for range 2 {
			agent := agentmodel.NewAgent(uuid.New())
			agent.Status.Connected = true
			agent.Status.LastReportedAt = time.Now()
			agent.Status.LastReportedTo = "target"
			require.NoError(t, producer.NotifyAgentUpdated(t.Context(), agent))
			require.NoError(t, producer.NotifyAgentUpdated(t.Context(), agent))
			ctx, cancel := context.WithCancel(t.Context())

			done := make(chan error, 1)
			go func() { done <- consumer.Run(ctx) }()

			select {
			case message := <-sent:
				assert.Equal(t, testServerID, message.Source)
				assert.Equal(t, "target", message.Target)
				assert.Equal(t, []uuid.UUID{agent.Metadata.InstanceUID},
					message.Payload.TargetAgentInstanceUIDs)
			case <-time.After(time.Second):
				cancel()
				t.Fatal("queued notification was not dispatched")
			}

			cancel()
			require.NoError(t, <-done)
		}

		sender.AssertExpectations(t)
	})
}
