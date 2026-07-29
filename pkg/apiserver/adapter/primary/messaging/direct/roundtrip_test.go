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

const targetServerID = "server-b"

var _ agentport.ServerEventSenderPort = (*outdirect.EventSenderAdapter)(nil)

type transport struct {
	name        string
	newReceiver func(addr, serverID, token string, logger *slog.Logger) indirect.Receiver
	newDelivery func(token string, logger *slog.Logger) outdirect.Deliverer
}

func transports() []transport {
	return []transport{
		{
			name: "http",
			newReceiver: func(addr, serverID, token string, logger *slog.Logger) indirect.Receiver {
				return indirect.NewHTTPReceiver(addr, serverID, token, logger)
			},
			newDelivery: func(token string, logger *slog.Logger) outdirect.Deliverer {
				return outdirect.NewHTTPDeliverer(token, logger)
			},
		},
		{
			name: "grpc",
			newReceiver: func(addr, serverID, token string, logger *slog.Logger) indirect.Receiver {
				return indirect.NewGRPCReceiver(addr, serverID, token, logger)
			},
			newDelivery: func(token string, logger *slog.Logger) outdirect.Deliverer {
				return outdirect.NewGRPCDeliverer(token, logger)
			},
		},
	}
}

func peer(address string) *agentmodel.Server {
	return &agentmodel.Server{
		ID:              targetServerID,
		Address:         address,
		LastHeartbeatAt: time.Time{},
		Conditions:      nil,
	}
}

func sampleMessage() serverevent.Message {
	return serverevent.Message{
		Source: "server-a",
		Target: targetServerID,
		Type:   serverevent.MessageTypeSendServerToAgent,
		Payload: serverevent.MessagePayload{
			MessageForServerToAgent: &serverevent.MessageForServerToAgent{
				TargetAgentInstanceUIDs: []uuid.UUID{uuid.New()},
			},
			MessageForInvalidateAgentCache: nil,
		},
	}
}

// TestDirectTransport_RoundTrip exercises a full sender -> receiver delivery, including
// bearer-token authentication, over both sub-protocols.
func TestDirectTransport_RoundTrip(t *testing.T) {
	t.Parallel()

	const token = "shared-secret"

	for _, tt := range transports() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := slog.New(slog.NewTextHandler(testutil.TestLogWriter{T: t}, nil))
			address := freeAddress(t)

			received := make(chan *serverevent.Message, 1)

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			receiver := indirect.NewEventReceiverAdapter(tt.newReceiver(address, targetServerID, token, logger), logger)
			serveErr := make(chan error, 1)

			go func() { serveErr <- receiver.StartReceiver(ctx, capturingHandler(received)) }()

			deliverer := tt.newDelivery(token, logger)

			defer func() { _ = deliverer.Close() }()

			sender := outdirect.NewEventSenderAdapter(deliverer, logger)
			message := sampleMessage()

			sendUntilDelivered(ctx, t, sender, peer(address), message)

			got := requireReceive(t, received)
			assert.Equal(t, message.Source, got.Source)
			assert.Equal(t, message.Type, got.Type)
			require.NotNil(t, got.Payload.MessageForServerToAgent)

			cancel()
			require.NoError(t, <-serveErr)
		})
	}
}

// TestDirectTransport_RejectsBadToken verifies the receiver refuses a sender that presents
// the wrong credential, and never invokes the handler.
func TestDirectTransport_RejectsBadToken(t *testing.T) {
	t.Parallel()

	for _, tt := range transports() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := slog.New(slog.NewTextHandler(testutil.TestLogWriter{T: t}, nil))
			address := freeAddress(t)

			received := make(chan *serverevent.Message, 1)

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			receiver := indirect.NewEventReceiverAdapter(
				tt.newReceiver(address, targetServerID, "right-token", logger), logger)
			serveErr := make(chan error, 1)

			go func() { serveErr <- receiver.StartReceiver(ctx, capturingHandler(received)) }()

			deliverer := tt.newDelivery("wrong-token", logger)

			defer func() { _ = deliverer.Close() }()

			sender := outdirect.NewEventSenderAdapter(deliverer, logger)

			requireServing(ctx, t, address)

			err := sender.SendMessageToServer(ctx, peer(address), sampleMessage())
			require.Error(t, err)
			requireNoReceive(t, received)

			cancel()
			require.NoError(t, <-serveErr)
		})
	}
}

// TestDirectTransport_RejectsMissingToken verifies the receiver refuses a sender that
// presents no credential at all when a token is required.
func TestDirectTransport_RejectsMissingToken(t *testing.T) {
	t.Parallel()

	for _, tt := range transports() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := slog.New(slog.NewTextHandler(testutil.TestLogWriter{T: t}, nil))
			address := freeAddress(t)

			received := make(chan *serverevent.Message, 1)

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			receiver := indirect.NewEventReceiverAdapter(
				tt.newReceiver(address, targetServerID, "required-token", logger), logger)
			serveErr := make(chan error, 1)

			go func() { serveErr <- receiver.StartReceiver(ctx, capturingHandler(received)) }()

			// Sender configured with no token, so it presents no credential.
			deliverer := tt.newDelivery("", logger)

			defer func() { _ = deliverer.Close() }()

			sender := outdirect.NewEventSenderAdapter(deliverer, logger)

			requireServing(ctx, t, address)

			err := sender.SendMessageToServer(ctx, peer(address), sampleMessage())
			require.Error(t, err)
			requireNoReceive(t, received)

			cancel()
			require.NoError(t, <-serveErr)
		})
	}
}

// TestDirectTransport_RejectsWrongTarget verifies a message addressed to a different server
// is refused rather than processed by whoever received it.
func TestDirectTransport_RejectsWrongTarget(t *testing.T) {
	t.Parallel()

	for _, tt := range transports() {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := slog.New(slog.NewTextHandler(testutil.TestLogWriter{T: t}, nil))
			address := freeAddress(t)

			received := make(chan *serverevent.Message, 1)

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			// Receiver believes it is "server-c"; the message targets "server-b".
			receiver := indirect.NewEventReceiverAdapter(tt.newReceiver(address, "server-c", "", logger), logger)
			serveErr := make(chan error, 1)

			go func() { serveErr <- receiver.StartReceiver(ctx, capturingHandler(received)) }()

			deliverer := tt.newDelivery("", logger)

			defer func() { _ = deliverer.Close() }()

			sender := outdirect.NewEventSenderAdapter(deliverer, logger)

			requireServing(ctx, t, address)

			err := sender.SendMessageToServer(ctx, peer(address), sampleMessage())
			require.Error(t, err)
			requireNoReceive(t, received)

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
	deliverer := outdirect.NewHTTPDeliverer("", logger)

	defer func() { _ = deliverer.Close() }()

	sender := outdirect.NewEventSenderAdapter(deliverer, logger)

	err := sender.SendMessageToServer(t.Context(), peer(""), sampleMessage())
	require.ErrorIs(t, err, outdirect.ErrNoPeerAddress)
}

func capturingHandler(received chan<- *serverevent.Message) agentport.ReceiveServerEventHandler {
	return func(_ context.Context, msg *serverevent.Message) error {
		received <- msg

		return nil
	}
}

// requireReceive returns the first message delivered to the channel, failing if none
// arrives within the timeout.
func requireReceive(t *testing.T, received <-chan *serverevent.Message) *serverevent.Message {
	t.Helper()

	var got *serverevent.Message

	require.Eventually(t, func() bool {
		select {
		case got = <-received:
			return true
		default:
			return false
		}
	}, 5*time.Second, 50*time.Millisecond, "timed out waiting for message delivery")

	return got
}

// requireNoReceive asserts that no message is delivered to the channel.
func requireNoReceive(t *testing.T, received <-chan *serverevent.Message) {
	t.Helper()

	require.Never(t, func() bool {
		return len(received) > 0
	}, 200*time.Millisecond, 20*time.Millisecond, "handler was invoked unexpectedly")
}

// sendUntilDelivered retries delivery until it succeeds, tolerating the brief window
// before the receiver is accepting connections.
func sendUntilDelivered(
	ctx context.Context,
	t *testing.T,
	sender *outdirect.EventSenderAdapter,
	server *agentmodel.Server,
	message serverevent.Message,
) {
	t.Helper()

	require.Eventually(t, func() bool {
		return sender.SendMessageToServer(ctx, server, message) == nil
	}, 5*time.Second, 50*time.Millisecond, "failed to deliver message before deadline")
}

// requireServing blocks until the receiver at address accepts a TCP connection, so a
// subsequent send exercises the receiver rather than a connection-refused error.
func requireServing(ctx context.Context, t *testing.T, address string) {
	t.Helper()

	require.Eventually(t, func() bool {
		var dialer net.Dialer

		conn, err := dialer.DialContext(ctx, "tcp", address)
		if err != nil {
			return false
		}

		_ = conn.Close()

		return true
	}, 5*time.Second, 50*time.Millisecond, "receiver never started serving on %s", address)
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
