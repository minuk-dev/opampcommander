package direct_test

import (
	"context"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	indirect "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/messaging/direct"
	outdirect "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/messaging/direct"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

// staticResolver resolves every server ID to a single fixed address.
type staticResolver struct {
	address string
}

func (r staticResolver) GetServer(_ context.Context, id string) (*agentmodel.Server, error) {
	return &agentmodel.Server{
		ID:              id,
		Address:         r.address,
		LastHeartbeatAt: time.Time{},
		Conditions:      nil,
	}, nil
}

func TestDirectTransport_RoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		newReceiver func(addr string, logger *slog.Logger) indirect.Receiver
		newDelivery func(logger *slog.Logger) outdirect.Deliverer
	}{
		{
			name: "http",
			newReceiver: func(addr string, logger *slog.Logger) indirect.Receiver {
				return indirect.NewHTTPReceiver(addr, logger)
			},
			newDelivery: func(logger *slog.Logger) outdirect.Deliverer {
				return outdirect.NewHTTPDeliverer(logger)
			},
		},
		{
			name: "grpc",
			newReceiver: func(addr string, logger *slog.Logger) indirect.Receiver {
				return indirect.NewGRPCReceiver(addr, logger)
			},
			newDelivery: func(logger *slog.Logger) outdirect.Deliverer {
				return outdirect.NewGRPCDeliverer(logger)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := slog.New(slog.NewTextHandler(testutil.TestLogWriter{T: t}, nil))
			address := freeAddress(t)

			received := make(chan *serverevent.Message, 1)
			//nolint:unparam // signature must match agentport.ReceiveServerEventHandler.
			handler := func(_ context.Context, msg *serverevent.Message) error {
				received <- msg

				return nil
			}

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			receiver := indirect.NewEventReceiverAdapter(tt.newReceiver(address, logger), logger)
			serveErr := make(chan error, 1)

			go func() { serveErr <- receiver.StartReceiver(ctx, handler) }()

			deliverer := tt.newDelivery(logger)

			defer func() { _ = deliverer.Close() }()

			sender := outdirect.NewEventSenderAdapter(staticResolver{address: address}, deliverer, logger)

			agentUID := uuid.New()
			message := serverevent.Message{
				Source: "server-a",
				Target: "server-b",
				Type:   serverevent.MessageTypeSendServerToAgent,
				Payload: serverevent.MessagePayload{
					MessageForServerToAgent: &serverevent.MessageForServerToAgent{
						TargetAgentInstanceUIDs: []uuid.UUID{agentUID},
					},
					MessageForInvalidateAgentCache: nil,
				},
			}

			sendUntilDelivered(ctx, t, sender, message)

			select {
			case got := <-received:
				assert.Equal(t, message.Source, got.Source)
				assert.Equal(t, message.Type, got.Type)
				require.NotNil(t, got.Payload.MessageForServerToAgent)
				assert.Equal(t, []uuid.UUID{agentUID}, got.Payload.TargetAgentInstanceUIDs)
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for message delivery")
			}

			cancel()
			require.NoError(t, <-serveErr)
		})
	}
}

// TestDirectSender_NoAddress verifies that a peer without a registered address is a
// clear error rather than a silent drop.
func TestDirectSender_NoAddress(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(testutil.TestLogWriter{T: t}, nil))
	deliverer := outdirect.NewHTTPDeliverer(logger)

	defer func() { _ = deliverer.Close() }()

	sender := outdirect.NewEventSenderAdapter(staticResolver{address: ""}, deliverer, logger)

	err := sender.SendMessageToServer(t.Context(), "server-b", serverevent.Message{
		Source: "server-a",
		Target: "server-b",
		Type:   serverevent.MessageTypeInvalidateAgentCache,
		Payload: serverevent.MessagePayload{
			MessageForServerToAgent: nil,
			MessageForInvalidateAgentCache: &serverevent.MessageForInvalidateAgentCache{
				AgentInstanceUIDs: []uuid.UUID{uuid.New()},
			},
		},
	})
	require.ErrorIs(t, err, outdirect.ErrNoPeerAddress)
}

var _ agentport.ServerEventSenderPort = (*outdirect.EventSenderAdapter)(nil)

// sendUntilDelivered retries delivery until it succeeds, tolerating the brief window
// before the receiver is accepting connections.
func sendUntilDelivered(
	ctx context.Context,
	t *testing.T,
	sender *outdirect.EventSenderAdapter,
	message serverevent.Message,
) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)

	for {
		err := sender.SendMessageToServer(ctx, "server-b", message)
		if err == nil {
			return
		}

		if time.Now().After(deadline) {
			t.Fatalf("failed to deliver message before deadline: %v", err)
		}

		time.Sleep(50 * time.Millisecond)
	}
}

// freeAddress reserves an ephemeral localhost port and returns its address.
func freeAddress(t *testing.T) string {
	t.Helper()

	var listenConfig net.ListenConfig

	listener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	address := listener.Addr().String()
	require.NoError(t, listener.Close())

	return address
}
