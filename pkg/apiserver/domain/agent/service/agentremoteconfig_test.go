package agentservice_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// arcFakePersistence is a minimal in-memory AgentRemoteConfigPersistencePort for
// the lifecycle tests. It records the last Put and serves a single stored config.
type arcFakePersistence struct {
	stored   *agentmodel.AgentRemoteConfig
	getErr   error
	putCalls int
	lastPut  *agentmodel.AgentRemoteConfig
}

func (f *arcFakePersistence) GetAgentRemoteConfig(
	_ context.Context, _ string, _ string, _ *model.GetOptions,
) (*agentmodel.AgentRemoteConfig, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}

	return f.stored, nil
}

func (f *arcFakePersistence) PutAgentRemoteConfig(
	_ context.Context, config *agentmodel.AgentRemoteConfig,
) (*agentmodel.AgentRemoteConfig, error) {
	f.putCalls++
	f.lastPut = config

	return config, nil
}

func (f *arcFakePersistence) ListAgentRemoteConfigs(
	_ context.Context, _ *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentRemoteConfig], error) {
	return &model.ListResponse[*agentmodel.AgentRemoteConfig]{}, nil
}

var _ agentport.AgentRemoteConfigPersistencePort = (*arcFakePersistence)(nil)

func TestAgentRemoteConfigService_CreateAgentRemoteConfig_Stamps(t *testing.T) {
	t.Parallel()

	persistence := &arcFakePersistence{}
	// Create/Update only touch persistence + clock; the reconcile collaborators are unused here.
	svc := agentservice.NewAgentRemoteConfigService(persistence, nil, nil)

	input := &agentmodel.AgentRemoteConfig{
		Metadata: agentmodel.AgentRemoteConfigMetadata{Name: "cfg", Namespace: "default"},
		Spec:     agentmodel.AgentRemoteConfigSpec{Value: []byte("a"), ContentType: "text/yaml"},
	}

	created, err := svc.CreateAgentRemoteConfig(t.Context(), input, "tester")

	require.NoError(t, err)
	assert.Equal(t, 1, persistence.putCalls)
	require.NotEmpty(t, created.Status.Conditions, "creation must record a condition")

	cond := created.Status.Conditions[0]
	assert.Equal(t, model.ConditionTypeCreated, cond.Type)
	assert.Equal(t, "tester", cond.Reason, "the acting user must be stamped as the condition reason")
}

func TestAgentRemoteConfigService_UpdateAgentRemoteConfig_PreservesImmutableFields(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	stored := &agentmodel.AgentRemoteConfig{
		Metadata: agentmodel.AgentRemoteConfigMetadata{Name: "cfg", Namespace: "default", CreatedAt: createdAt},
		Spec:     agentmodel.AgentRemoteConfigSpec{Value: []byte("old"), ContentType: "text/yaml"},
		Status: agentmodel.AgentRemoteConfigResourceStatus{
			Conditions: []model.Condition{{Type: model.ConditionTypeCreated}},
		},
	}

	persistence := &arcFakePersistence{stored: stored}
	svc := agentservice.NewAgentRemoteConfigService(persistence, nil, nil)

	incoming := &agentmodel.AgentRemoteConfig{
		Metadata: agentmodel.AgentRemoteConfigMetadata{
			Name:      "cfg",
			Namespace: "default",
			CreatedAt: time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		Spec: agentmodel.AgentRemoteConfigSpec{Value: []byte("new"), ContentType: "text/yaml"},
	}

	updated, err := svc.UpdateAgentRemoteConfig(t.Context(), "default", "cfg", incoming)

	require.NoError(t, err)
	assert.Equal(t, createdAt, updated.Metadata.CreatedAt, "CreatedAt must be preserved from the stored config")
	assert.Equal(t, []byte("new"), updated.Spec.Value, "mutable spec must be applied")
	assert.NotEmpty(t, updated.Status.Conditions, "existing lifecycle conditions must be preserved")
}
