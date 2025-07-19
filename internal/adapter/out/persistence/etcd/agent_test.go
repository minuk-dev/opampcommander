package etcd_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/etcd"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	etcdTestContainer "github.com/testcontainers/testcontainers-go/modules/etcd"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func TestAgentEtcdAdapter_GetAgent(t *testing.T) {
	t.Parallel()
	base := testutil.NewBase(t)
	ctx := t.Context()
	etcdContainer, err := etcdTestContainer.Run(ctx, "gcr.io/etcd-development/etcd:v3.5.14")
	require.NoError(t, err)

	etcdEndpoint, err := etcdContainer.ClientEndpoint(ctx)
	require.NoError(t, err)

	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints: []string{etcdEndpoint},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		err := etcdClient.Close()
		require.NoError(t, err)
	})

	agentEtcdAdapter := etcd.NewAgentEtcdAdapter(etcdClient, base.Logger) // Assuming NewAgentEtcdAdapter is defined

	t.Run("Happy case", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		instanceUID := uuid.New()
		// given
		_, err := etcdClient.Put(ctx, "agents/"+instanceUID.String(), `
		{
			"instanceUID": "`+instanceUID.String()+`",
			"capabilities": null,
			"description": null,
			"effectiveConfig": null,
			"packageStatuses": null,
			"componentHealth": null,
			"remoteConfig": {},
			"customCapabilities": null,
			"availableComponents": null,
			"reportFullState": false
		}`)
		require.NoError(t, err)

		// when
		agent, err := agentEtcdAdapter.GetAgent(ctx, instanceUID)

		// then
		assert.NoError(t, err)
		assert.Equal(t, instanceUID, agent.InstanceUID)
	})

	t.Run("Agent not found", func(t *testing.T) {
		t.Parallel()
		// Test case for when the agent is not found in etcd
	})

	t.Run("Multiple agents found", func(t *testing.T) {
		t.Parallel()
		// Test case for when multiple agents are found in etcd
	})

	t.Run("Etcd error", func(t *testing.T) {
		t.Parallel()
		// Test case for when there is an error communicating with etcd
	})
}

func TestAgentEtcdAdapter_ListAgents(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	etcdContainer, err := etcdTestContainer.Run(ctx, "gcr.io/etcd-development/etcd:v3.5.14")
	require.NoError(t, err)

	etcdEndpoint, err := etcdContainer.ClientEndpoint(ctx)
	require.NoError(t, err)

	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints: []string{etcdEndpoint},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		err := etcdClient.Close()
		require.NoError(t, err)
	})

	t.Run("Happy case", func(t *testing.T) {
		t.Parallel()
	})

	t.Run("No agents found", func(t *testing.T) {
		t.Parallel()
		// Test case for when no agents are found in etcd
	})

	t.Run("Etcd error", func(t *testing.T) {
		t.Parallel()
		// Test case for when there is an error communicating with etcd
	})
}
