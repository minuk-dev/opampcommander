package agentservice_test

import (
	"context"
	"log/slog"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

type notificationServerUsecase struct{ agentport.ServerUsecase }

func (notificationServerUsecase) GetServer(context.Context, string) (*agentmodel.Server, error) {
	return &agentmodel.Server{ID: "remote"}, nil
}

type notificationMessageUsecase struct {
	agentport.ServerMessageUsecase

	calls  int
	agents int
	err    error
}

func (s *notificationMessageUsecase) SendMessageToServer(
	_ context.Context, _ *agentmodel.Server, message serverevent.Message,
) error {
	s.calls++
	s.agents = len(message.Payload.TargetAgentInstanceUIDs)

	return s.err
}

func TestAgentNotificationService_BestEffortCoalescedSend(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"success", "transport failure"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				identity := new(MockServerIdentityProvider)
				identity.On("CurrentServerID").Return("local")

				messages := &notificationMessageUsecase{}
				if name == "transport failure" {
					messages.err = model.ErrTargetServerUnreachable
				}

				svc := agentservice.NewAgentNotificationService(messages, notificationServerUsecase{}, identity,
					slog.New(slog.DiscardHandler))
				for range 2 {
					updated := agentmodel.NewAgent(uuid.New())
					updated.Status.LastReportedAt = time.Now()
					updated.Status.LastReportedTo = "remote"
					require.NoError(t, svc.NotifyAgentUpdated(t.Context(), updated))
				}

				require.Zero(t, messages.calls, "notifications must not wait for transport delivery")

				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()

				done := make(chan struct{})
				go func() { defer close(done); _ = svc.Run(ctx) }()

				synctest.Wait()
				time.Sleep(agentservice.DefaultNotificationFlushInterval)
				synctest.Wait()
				cancel()
				<-done
				require.Equal(t, 1, messages.calls, "notifications should remain coalesced")
				require.Equal(t, 2, messages.agents)
			})
		})
	}
}
