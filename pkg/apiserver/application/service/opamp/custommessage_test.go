//nolint:testpackage // white-box test of the unexported custom-message dispatch path
package opamp

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
)

var errHandlerBoom = errors.New("boom")

// fakeCustomMessageHandler is a configurable CustomMessageHandler: it records the inbound
// message it received and returns the preconfigured reply/err.
type fakeCustomMessageHandler struct {
	capability string
	reply      *CustomMessage
	err        error
	gotMsg     *CustomMessage
}

func (h *fakeCustomMessageHandler) Capability() string { return h.capability }

func (h *fakeCustomMessageHandler) HandleCustomMessage(
	_ context.Context, _ *agentmodel.Agent, msg *CustomMessage,
) (*CustomMessage, error) {
	h.gotMsg = msg

	return h.reply, h.err
}

func TestNewCustomMessageRegistry(t *testing.T) {
	t.Parallel()

	t.Run("indexes handlers and advertises sorted capabilities", func(t *testing.T) {
		t.Parallel()

		hb := &fakeCustomMessageHandler{capability: "com.example.b"}
		ha := &fakeCustomMessageHandler{capability: "com.example.a"}

		reg, err := NewCustomMessageRegistry([]CustomMessageHandler{hb, ha})
		require.NoError(t, err)

		// Sorted regardless of registration order.
		assert.Equal(t, []string{"com.example.a", "com.example.b"}, []string(reg.Capabilities()))

		got, ok := reg.Handler("com.example.a")
		require.True(t, ok)
		assert.Same(t, ha, got)

		_, ok = reg.Handler("com.example.missing")
		assert.False(t, ok)
	})

	t.Run("empty registry advertises no capabilities", func(t *testing.T) {
		t.Parallel()

		reg, err := NewCustomMessageRegistry(nil)
		require.NoError(t, err)
		assert.Empty(t, reg.Capabilities())
	})

	t.Run("duplicate capability is a wiring error", func(t *testing.T) {
		t.Parallel()

		_, err := NewCustomMessageRegistry([]CustomMessageHandler{
			&fakeCustomMessageHandler{capability: "com.example.dup"},
			&fakeCustomMessageHandler{capability: "com.example.dup"},
		})
		require.ErrorIs(t, err, ErrDuplicateCustomCapabilityHandler)
	})
}

// agentWithCustomCapabilities builds an agent advertising the given custom capabilities.
func agentWithCustomCapabilities(capabilities ...string) *agentmodel.Agent {
	return agentmodel.NewAgent(uuid.New(),
		agentmodel.WithCustomCapabilities(&agentmodel.AgentCustomCapabilities{Capabilities: capabilities}))
}

func mustRegistry(t *testing.T, handlers ...CustomMessageHandler) *CustomMessageRegistry {
	t.Helper()

	reg, err := NewCustomMessageRegistry(handlers)
	require.NoError(t, err)

	return reg
}

func serviceWithRegistry(reg *CustomMessageRegistry) *Service {
	return &Service{
		logger:                slog.New(slog.DiscardHandler),
		customMessageRegistry: reg,
	}
}

func TestHandleInboundCustomMessage(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)

	t.Run("routes to the handler and returns its reply when the agent advertised it", func(t *testing.T) {
		t.Parallel()

		handler := &fakeCustomMessageHandler{
			capability: "com.example.cap",
			reply:      &CustomMessage{Capability: "com.example.cap", Type: "pong", Data: []byte("out")},
		}
		svc := serviceWithRegistry(mustRegistry(t, handler))
		agent := agentWithCustomCapabilities("com.example.cap")

		reply := svc.handleInboundCustomMessage(t.Context(), logger, agent,
			&protobufs.CustomMessage{Capability: "com.example.cap", Type: "ping", Data: []byte("in")})

		require.NotNil(t, reply)
		assert.Equal(t, "com.example.cap", reply.GetCapability())
		assert.Equal(t, "pong", reply.GetType())
		assert.Equal(t, []byte("out"), reply.GetData())

		// The handler saw the decoded inbound message.
		require.NotNil(t, handler.gotMsg)
		assert.Equal(t, "ping", handler.gotMsg.Type)
		assert.Equal(t, []byte("in"), handler.gotMsg.Data)
	})

	t.Run("capability mismatch: reply is dropped when the agent did not advertise it", func(t *testing.T) {
		t.Parallel()

		handler := &fakeCustomMessageHandler{
			capability: "com.example.cap",
			reply:      &CustomMessage{Capability: "com.example.other", Type: "pong"},
		}
		svc := serviceWithRegistry(mustRegistry(t, handler))
		// Agent advertised only the inbound capability, not the reply's capability.
		agent := agentWithCustomCapabilities("com.example.cap")

		reply := svc.handleInboundCustomMessage(t.Context(), logger, agent,
			&protobufs.CustomMessage{Capability: "com.example.cap"})

		assert.Nil(t, reply, "must not send a custom message for an unadvertised capability")
	})

	t.Run("no handler for the capability drops the message", func(t *testing.T) {
		t.Parallel()

		svc := serviceWithRegistry(mustRegistry(t))
		agent := agentWithCustomCapabilities("com.example.cap")

		reply := svc.handleInboundCustomMessage(t.Context(), logger, agent,
			&protobufs.CustomMessage{Capability: "com.example.cap"})

		assert.Nil(t, reply)
	})

	t.Run("handler error yields no reply", func(t *testing.T) {
		t.Parallel()

		handler := &fakeCustomMessageHandler{capability: "com.example.cap", err: errHandlerBoom}
		svc := serviceWithRegistry(mustRegistry(t, handler))
		agent := agentWithCustomCapabilities("com.example.cap")

		reply := svc.handleInboundCustomMessage(t.Context(), logger, agent,
			&protobufs.CustomMessage{Capability: "com.example.cap"})

		assert.Nil(t, reply)
	})

	t.Run("handler with no reply yields no reply", func(t *testing.T) {
		t.Parallel()

		handler := &fakeCustomMessageHandler{capability: "com.example.cap", reply: nil}
		svc := serviceWithRegistry(mustRegistry(t, handler))
		agent := agentWithCustomCapabilities("com.example.cap")

		reply := svc.handleInboundCustomMessage(t.Context(), logger, agent,
			&protobufs.CustomMessage{Capability: "com.example.cap"})

		assert.Nil(t, reply)
	})

	t.Run("nil message is a no-op", func(t *testing.T) {
		t.Parallel()

		svc := serviceWithRegistry(mustRegistry(t))
		assert.Nil(t, svc.handleInboundCustomMessage(t.Context(), logger, agentWithCustomCapabilities(), nil))
	})

	t.Run("nil registry is a no-op", func(t *testing.T) {
		t.Parallel()

		svc := &Service{logger: logger}
		reply := svc.handleInboundCustomMessage(t.Context(), logger, agentWithCustomCapabilities("com.example.cap"),
			&protobufs.CustomMessage{Capability: "com.example.cap"})
		assert.Nil(t, reply)
	})
}
