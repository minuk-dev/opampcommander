package agentservice_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// fakeARCPersistence is a minimal AgentRemoteConfigPersistencePort returning one stored config.
type fakeARCPersistence struct {
	stored *agentmodel.AgentRemoteConfig
}

func (f *fakeARCPersistence) GetAgentRemoteConfig(
	_ context.Context, namespace, name string, _ *model.GetOptions,
) (*agentmodel.AgentRemoteConfig, error) {
	if f.stored == nil ||
		f.stored.Metadata.Namespace != namespace ||
		f.stored.Metadata.Name != name {
		return nil, model.ErrResourceNotExist
	}

	return f.stored, nil
}

func (f *fakeARCPersistence) PutAgentRemoteConfig(
	_ context.Context, config *agentmodel.AgentRemoteConfig,
) (*agentmodel.AgentRemoteConfig, error) {
	f.stored = config

	return config, nil
}

func (f *fakeARCPersistence) ListAgentRemoteConfigs(
	_ context.Context, _ *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentRemoteConfig], error) {
	return &model.ListResponse[*agentmodel.AgentRemoteConfig]{Items: nil, Continue: "", RemainingItemCount: 0}, nil
}

// spyEndpointDetection records whether endpoint detection ran for a remote config.
type spyEndpointDetection struct {
	reconciled *agentmodel.AgentRemoteConfig
}

func (s *spyEndpointDetection) ReconcileEndpointsFromRemoteConfig(
	_ context.Context, remoteConfig *agentmodel.AgentRemoteConfig,
) error {
	s.reconciled = remoteConfig

	return nil
}

func (s *spyEndpointDetection) ExtractEndpointsFromAgent(
	*agentmodel.Agent,
) ([]*agentmodel.Endpoint, error) {
	return nil, nil
}

// spyAgentGroup records the propagation call; the embedded interface covers the rest.
type spyAgentGroup struct {
	agentport.AgentGroupUsecase

	propagatedNamespace string
	propagatedName      string
}

func (s *spyAgentGroup) PropagateAgentRemoteConfigChange(
	_ context.Context, namespace, remoteConfigName string,
) error {
	s.propagatedNamespace = namespace
	s.propagatedName = remoteConfigName

	return nil
}

func TestAgentRemoteConfigService_ReconcileAgentRemoteConfig(t *testing.T) {
	t.Parallel()

	t.Run("runs endpoint detection and group propagation for an existing config", func(t *testing.T) {
		t.Parallel()

		//exhaustruct:ignore
		stored := &agentmodel.AgentRemoteConfig{
			Metadata: agentmodel.AgentRemoteConfigMetadata{Namespace: "default", Name: "obs"},
		}
		persistence := &fakeARCPersistence{stored: stored}
		detection := &spyEndpointDetection{}
		group := &spyAgentGroup{}

		service := agentservice.NewAgentRemoteConfigService(persistence, detection, group)

		err := service.ReconcileAgentRemoteConfig(context.Background(), "default", "obs")

		require.NoError(t, err)
		assert.Same(t, stored, detection.reconciled, "endpoint detection should run for the loaded config")
		assert.Equal(t, "default", group.propagatedNamespace)
		assert.Equal(t, "obs", group.propagatedName)
	})

	t.Run("returns an error when the config does not exist", func(t *testing.T) {
		t.Parallel()

		persistence := &fakeARCPersistence{stored: nil}
		detection := &spyEndpointDetection{}
		group := &spyAgentGroup{}

		service := agentservice.NewAgentRemoteConfigService(persistence, detection, group)

		err := service.ReconcileAgentRemoteConfig(context.Background(), "default", "missing")

		require.Error(t, err)
		assert.Nil(t, detection.reconciled, "endpoint detection must not run when the config is missing")
		assert.Empty(t, group.propagatedName, "propagation must not run when the config is missing")
	})
}
