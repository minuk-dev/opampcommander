package inmemory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/store/inmemory"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
)

func TestAgentGroupChangeStore_BoundedQueueAndCancellation(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	store := inmemory.NewAgentGroupChangeStore(1)
	change := agentport.AgentGroupChange{Namespace: "ns", Name: "group"}
	require.True(t, store.TryEnqueue(change))
	change.Name = "caller-mutation"
	assert.False(t, store.TryEnqueue(change))

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	require.ErrorIs(t, store.Enqueue(cancelled, change), context.Canceled)
	got, err := store.Next(ctx)
	require.NoError(t, err)
	assert.Equal(t, "group", got.Name)
	assert.Equal(t, "ns", got.Namespace)
	require.NoError(t, store.Enqueue(ctx, change))
	got, err = store.Next(ctx)
	require.NoError(t, err)
	assert.Equal(t, "caller-mutation", got.Name)

	_, err = store.Next(cancelled)
	require.ErrorIs(t, err, context.Canceled)
}
