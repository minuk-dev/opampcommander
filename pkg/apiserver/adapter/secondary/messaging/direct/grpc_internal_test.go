package direct

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/serverevent"
)

// TestGRPCDeliverer_ConnCachingAndEvict exercises the lazy per-address connection cache
// and its eviction directly, since grpc.NewClient does not dial until first use.
func TestGRPCDeliverer_ConnCachingAndEvict(t *testing.T) {
	t.Parallel()

	deliverer := NewGRPCDeliverer("", slog.New(slog.DiscardHandler))

	defer func() { _ = deliverer.Close() }()

	const address = "127.0.0.1:65500"

	first, err := deliverer.connFor(address)
	require.NoError(t, err)

	second, err := deliverer.connFor(address)
	require.NoError(t, err)

	assert.Same(t, first, second, "connFor must reuse the cached connection")
	assert.Len(t, deliverer.conns, 1)

	deliverer.evict(address)
	assert.Empty(t, deliverer.conns, "evict must remove the cached connection")

	// Evicting an absent address is a no-op.
	deliverer.evict(address)
	assert.Empty(t, deliverer.conns)
}

// TestGRPCDeliverer_DeliverToDeadPeerEvicts verifies that a failed RPC surfaces an error
// and drops the cached connection so the next send re-dials. grpc.NewClient does not dial
// until the RPC runs, so a short deadline against an unreachable address fails deterministically.
func TestGRPCDeliverer_DeliverToDeadPeerEvicts(t *testing.T) {
	t.Parallel()

	deliverer := NewGRPCDeliverer("secret-token", slog.New(slog.DiscardHandler))

	defer func() { _ = deliverer.Close() }()

	const address = "127.0.0.1:65501"

	ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
	defer cancel()

	err := deliverer.Deliver(ctx, address, serverevent.Message{
		Source: "server-1",
		Target: "server-2",
		Type:   serverevent.MessageTypeInvalidateAgentCache,
		Payload: serverevent.MessagePayload{
			MessageForInvalidateAgentCache: &serverevent.MessageForInvalidateAgentCache{},
		},
	})
	require.Error(t, err)
	// The failed connection must be evicted so a later send re-dials.
	assert.Empty(t, deliverer.conns)
}

// TestGRPCDeliverer_CloseIsIdempotent verifies Close clears the connection cache and can be
// safely called again.
func TestGRPCDeliverer_CloseIsIdempotent(t *testing.T) {
	t.Parallel()

	deliverer := NewGRPCDeliverer("", slog.New(slog.DiscardHandler))

	_, err := deliverer.connFor("127.0.0.1:65502")
	require.NoError(t, err)
	require.Len(t, deliverer.conns, 1)

	require.NoError(t, deliverer.Close())
	assert.Empty(t, deliverer.conns)

	require.NoError(t, deliverer.Close())
}
