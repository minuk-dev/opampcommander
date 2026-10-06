package inmemory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/store/inmemory"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
)

func TestAgentGroupChangeStore_BoundedQueueAndCancellation(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	store := inmemory.NewAgentGroupChangeStore(1)
	change := agentport.AgentGroupChange{Namespace: "ns", Name: "group", Selector: agentmodel.AgentSelector{
		IdentifyingAttributes:    map[string]string{"service.name": "original"},
		NonIdentifyingAttributes: map[string]string{"version": "original"},
	}}
	require.True(t, store.TryEnqueue(change))
	change.Name = "caller-mutation"
	change.Selector.IdentifyingAttributes["service.name"] = "caller-mutation"
	change.Selector.NonIdentifyingAttributes["version"] = "caller-mutation"
	assert.False(t, store.TryEnqueue(change))

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	require.ErrorIs(t, store.Enqueue(cancelled, change), context.Canceled)
	got, err := store.Next(ctx)
	require.NoError(t, err)
	assert.Equal(t, "group", got.Name)
	assert.Equal(t, "ns", got.Namespace)
	assert.Equal(t, "original", got.Selector.IdentifyingAttributes["service.name"])
	assert.Equal(t, "original", got.Selector.NonIdentifyingAttributes["version"])
	require.NoError(t, store.Enqueue(ctx, change))
	change.Selector.IdentifyingAttributes["service.name"] = "later-mutation"
	got, err = store.Next(ctx)
	require.NoError(t, err)
	assert.Equal(t, "caller-mutation", got.Name)
	assert.Equal(t, "caller-mutation", got.Selector.IdentifyingAttributes["service.name"])

	_, err = store.Next(cancelled)
	require.ErrorIs(t, err, context.Canceled)
}
