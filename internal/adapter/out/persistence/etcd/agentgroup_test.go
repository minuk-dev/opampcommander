package etcd_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	etcdTestContainer "github.com/testcontainers/testcontainers-go/modules/etcd"
	"github.com/testcontainers/testcontainers-go/wait"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/etcd"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/agentgroup"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func setupAgentGroupEtcdAdapter(t *testing.T) (*clientv3.Client, *etcd.AgentGroupEtcdAdapter) {
	t.Helper()
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
	// cleanup
	// NOTE: container is cleaned by testcontainers automatically
	// we just close client
	return etcdClient, etcd.NewAgentGroupEtcdAdapter(etcdClient, testutil.NewBase(t).Logger)
}

func TestAgentGroupEtcdAdapter_PutAndGet(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()
	ctx := t.Context()
	client, adapter := setupAgentGroupEtcdAdapter(t)
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	agentGroup := agentgroup.New(
		"group-a",
		agentgroup.OfAttributes(map[string]string{"env": "prod", "team": "core"}),
		time.Now(),
		"tester",
	)

	// when
	err := adapter.PutAgentGroup(ctx, agentGroup)
	require.NoError(t, err)

	// then (direct etcd check)
	getResp, err := client.Get(ctx, "agentgroups/"+agentGroup.UID.String())
	require.NoError(t, err)
	assert.Equal(t, int64(1), getResp.Count)
	assert.NotEmpty(t, getResp.Kvs[0].Value)

	// load via adapter
	loaded, err := adapter.GetAgentGroup(ctx, agentGroup.UID)
	require.NoError(t, err)
	assert.Equal(t, agentGroup.UID, loaded.UID)
	assert.Equal(t, agentGroup.Name, loaded.Name)
	assert.Equal(t, agentGroup.Attributes, loaded.Attributes)
	assert.False(t, loaded.IsDeleted())
}

func TestAgentGroupEtcdAdapter_Get_NotFound(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()
	ctx := t.Context()
	client, adapter := setupAgentGroupEtcdAdapter(t)
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	nonExist := uuid.New()
	// when
	got, err := adapter.GetAgentGroup(ctx, nonExist)
	// then
	require.ErrorIs(t, err, port.ErrResourceNotExist)
	assert.Nil(t, got)
}

func TestAgentGroupEtcdAdapter_List(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()
	ctx := t.Context()
	client, adapter := setupAgentGroupEtcdAdapter(t)
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	// seed a few groups
	for range 3 {
		agentGroup := agentgroup.New(
			"group-"+uuid.NewString()[:8],
			agentgroup.OfAttributes(map[string]string{"idx": uuid.NewString()}),
			time.Now(),
			"tester",
		)
		require.NoError(t, adapter.PutAgentGroup(ctx, agentGroup))
	}

	// when
	resp, err := adapter.ListAgentGroups(ctx, nil)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.GreaterOrEqual(t, len(resp.Items), 3)
}

func TestAgentGroupEtcdAdapter_List_Pagination(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()
	ctx := t.Context()
	client, adapter := setupAgentGroupEtcdAdapter(t)
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	// seed groups
	for range 5 {
		agentGroup := agentgroup.New(
			"group-"+uuid.NewString()[:8],
			agentgroup.OfAttributes(map[string]string{"i": uuid.NewString()}),
			time.Now(),
			"tester",
		)
		require.NoError(t, adapter.PutAgentGroup(ctx, agentGroup))
	}

	// page 1
	resp1, err := adapter.ListAgentGroups(ctx, &model.ListOptions{Limit: 2, Continue: ""})
	require.NoError(t, err)
	assert.Len(t, resp1.Items, 2)
	assert.NotEmpty(t, resp1.Continue)

	// page 2
	resp2, err := adapter.ListAgentGroups(ctx, &model.ListOptions{Limit: 2, Continue: resp1.Continue})
	require.NoError(t, err)
	assert.Len(t, resp2.Items, 2)
	// continue token may or may not be empty depending on number of items left
}
