package direct

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
