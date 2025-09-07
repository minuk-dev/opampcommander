package etcd_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	etcdTestContainer "github.com/testcontainers/testcontainers-go/modules/etcd"
	"github.com/testcontainers/testcontainers-go/wait"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/etcd"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/remoteconfig"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestAgentEtcdAdapter_GetAgent(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()
	base := testutil.NewBase(t)
	ctx := t.Context()
	etcdContainer, err := etcdTestContainer.Run(
		ctx, "gcr.io/etcd-development/etcd:v3.5.14",
		testcontainers.WithWaitStrategy(
			wait.ForExposedPort(),
		),
	)
	require.NoError(t, err)

	etcdEndpoint, err := etcdContainer.ClientEndpoint(ctx)
	require.NoError(t, err)

	//exhaustruct:ignore
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
		require.NoError(t, err)
		assert.Equal(t, instanceUID, agent.InstanceUID)
	})

	t.Run("Agent not found", func(t *testing.T) {
		t.Parallel()

		notExistUID := uuid.New()
		// when
		agent, err := agentEtcdAdapter.GetAgent(ctx, notExistUID)
		// then
		require.ErrorIs(t, err, domainport.ErrResourceNotExist)
		assert.Nil(t, agent)
	})
}

func TestAgentEtcdAdapter_ListAgents(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()

	t.Run("Happy case", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		etcdContainer, err := etcdTestContainer.Run(
			ctx, "gcr.io/etcd-development/etcd:v3.5.14",
			testcontainers.WithWaitStrategy(
				wait.ForExposedPort(),
			),
		)
		require.NoError(t, err)

		etcdEndpoint, err := etcdContainer.ClientEndpoint(ctx)
		require.NoError(t, err)

		//exhaustruct:ignore
		etcdClient, err := clientv3.New(clientv3.Config{
			Endpoints: []string{etcdEndpoint},
		})
		require.NoError(t, err)
		t.Cleanup(func() {
			err := etcdClient.Close()
			require.NoError(t, err)
		})

		instanceUID := uuid.New()
		_, err = etcdClient.Put(ctx, "agents/"+instanceUID.String(), `
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

		agentEtcdAdapter := etcd.NewAgentEtcdAdapter(etcdClient, testutil.NewBase(t).Logger)

		// when
		listResponse, err := agentEtcdAdapter.ListAgents(ctx, nil)

		// then
		require.NoError(t, err)
		assert.NotNil(t, listResponse)
		// at least one agent should be present
		// because for better test performance, we uses shared etcd instance
		assert.GreaterOrEqual(t, len(listResponse.Items), 1)
	})
}

func TestAgentEtcdAdapter_PutAgent(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()

	t.Run("Happy case", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		etcdContainer, err := etcdTestContainer.Run(
			ctx, "gcr.io/etcd-development/etcd:v3.5.14",
			testcontainers.WithWaitStrategy(
				wait.ForExposedPort(),
			),
		)
		require.NoError(t, err)

		etcdEndpoint, err := etcdContainer.ClientEndpoint(ctx)
		require.NoError(t, err)

		//exhaustruct:ignore
		etcdClient, err := clientv3.New(clientv3.Config{
			Endpoints: []string{etcdEndpoint},
		})
		require.NoError(t, err)
		t.Cleanup(func() {
			err := etcdClient.Close()
			require.NoError(t, err)
		})

		instanceUID := uuid.New()
		agentEtcdAdapter := etcd.NewAgentEtcdAdapter(etcdClient, testutil.NewBase(t).Logger)

		// when
		err = agentEtcdAdapter.PutAgent(ctx, &model.Agent{
			InstanceUID:         instanceUID,
			Capabilities:        nil,
			Description:         nil,
			EffectiveConfig:     nil,
			PackageStatuses:     nil,
			ComponentHealth:     nil,
			RemoteConfig:        remoteconfig.New(),
			CustomCapabilities:  nil,
			AvailableComponents: nil,
			ReportFullState:     false,
		})
		require.NoError(t, err)

		// then
		getResponse, err := etcdClient.Get(ctx, "agents/"+instanceUID.String())
		require.NoError(t, err)
		assert.Equal(t, int64(1), getResponse.Count)
		assert.NotEmpty(t, getResponse.Kvs[0].Value)
	})
}

func TestAgentEtcdAdapter_ConfigShouldBeSameAfterSaveAndLoad(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()
	ctx := t.Context()
	etcdContainer, err := etcdTestContainer.Run(ctx,
		"gcr.io/etcd-development/etcd:v3.5.14",
		testcontainers.WithWaitStrategy(wait.ForExposedPort()))
	require.NoError(t, err)

	etcdEndpoint, err := etcdContainer.ClientEndpoint(ctx)
	require.NoError(t, err)

	//exhaustruct:ignore
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints: []string{etcdEndpoint},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		err := etcdClient.Close()
		require.NoError(t, err)
	})

	instanceUID := uuid.New()
	agentEtcdAdapter := etcd.NewAgentEtcdAdapter(etcdClient, testutil.NewBase(t).Logger)

	// when
	originalAgent := &model.Agent{
		InstanceUID:  instanceUID,
		Capabilities: nil,
		Description:  nil,
		EffectiveConfig: &model.AgentEffectiveConfig{
			ConfigMap: model.AgentConfigMap{
				ConfigMap: map[string]model.AgentConfigFile{
					"config.yaml": {
						Body:        []byte("key: value"),
						ContentType: "application/yaml",
					},
				},
			},
		},
		PackageStatuses:     nil,
		ComponentHealth:     nil,
		RemoteConfig:        remoteconfig.New(),
		CustomCapabilities:  nil,
		AvailableComponents: nil,
		ReportFullState:     false,
	}
	err = agentEtcdAdapter.PutAgent(ctx, originalAgent)
	require.NoError(t, err)
	loadedAgent, err := agentEtcdAdapter.GetAgent(ctx, instanceUID)
	require.NoError(t, err)

	// then
	assert.Equal(t, originalAgent.InstanceUID, loadedAgent.InstanceUID)
	assert.Equal(t, originalAgent.EffectiveConfig.ConfigMap.ConfigMap, loadedAgent.EffectiveConfig.ConfigMap.ConfigMap)
}
