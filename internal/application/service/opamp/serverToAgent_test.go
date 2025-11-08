//nolint:testpackage // Testing internal function buildRemoteConfig
package opamp

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/agent"
	"github.com/minuk-dev/opampcommander/internal/domain/model/agentgroup"
	"github.com/minuk-dev/opampcommander/internal/domain/model/vo"
)

// Mock AgentGroupUsecase for testing.
type mockAgentGroupUsecase struct {
	groups []*agentgroup.AgentGroup
	err    error
}

//nolint:nilnil // Mock method returns nil for both values when not implemented
func (m *mockAgentGroupUsecase) GetAgentGroup(_ context.Context, _ string) (*agentgroup.AgentGroup, error) {
	return nil, nil
}

//nolint:nilnil // Mock method returns nil for both values when not implemented
func (m *mockAgentGroupUsecase) ListAgentGroups(
	_ context.Context,
	_ *model.ListOptions,
) (*model.ListResponse[*agentgroup.AgentGroup], error) {
	return nil, nil
}

func (m *mockAgentGroupUsecase) SaveAgentGroup(_ context.Context, _ string, _ *agentgroup.AgentGroup) error {
	return nil
}

func (m *mockAgentGroupUsecase) DeleteAgentGroup(_ context.Context, _ string, _ time.Time, _ string) error {
	return nil
}

func (m *mockAgentGroupUsecase) GetAgentGroupsForAgent(
	_ context.Context,
	_ *model.Agent,
) ([]*agentgroup.AgentGroup, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.groups, nil
}

func TestBuildRemoteConfig_NoCapability(t *testing.T) {
	t.Parallel()

	// Given: Agent without AcceptsRemoteConfig capability
	logger := slog.Default()
	service := &Service{
		logger:            logger,
		agentGroupUsecase: &mockAgentGroupUsecase{},
	}

	agentModel := model.NewAgent(uuid.New())
	agentModel.Metadata.Capabilities = 0 // No capabilities

	// When: Build remote config
	result, err := service.buildRemoteConfig(context.Background(), agentModel)

	// Then: Should return nil
	require.NoError(t, err)
	assert.Nil(t, result, "Should not send config to agent without capability")
}

func TestBuildRemoteConfig_WithCapability_NoGroups(t *testing.T) {
	t.Parallel()

	// Given: Agent with AcceptsRemoteConfig capability but no matching groups
	logger := slog.Default()
	service := &Service{
		logger:            logger,
		agentGroupUsecase: &mockAgentGroupUsecase{groups: []*agentgroup.AgentGroup{}},
	}

	agentModel := model.NewAgent(uuid.New())
	agentModel.Metadata.Capabilities = agent.Capabilities(agent.AgentCapabilityAcceptsRemoteConfig)

	// When: Build remote config
	result, err := service.buildRemoteConfig(context.Background(), agentModel)

	// Then: Should return nil
	require.NoError(t, err)
	assert.Nil(t, result, "Should not send config when no groups exist")
}

func TestBuildRemoteConfig_WithCapability_WithConfig(t *testing.T) {
	t.Parallel()

	// Given: Agent with capability and matching group with config
	testConfig := "receivers:\n  otlp:\nexporters:\n  logging:\n"

	mockUsecase := &mockAgentGroupUsecase{
		groups: []*agentgroup.AgentGroup{
			{
				Name: "test-group",
				AgentConfig: &agentgroup.AgentConfig{
					Value: testConfig,
				},
			},
		},
	}

	logger := slog.Default()
	service := &Service{
		logger:            logger,
		agentGroupUsecase: mockUsecase,
	}

	agentModel := model.NewAgent(uuid.New())
	agentModel.Metadata.Capabilities = agent.Capabilities(agent.AgentCapabilityAcceptsRemoteConfig)

	// When: Build remote config
	result, err := service.buildRemoteConfig(context.Background(), agentModel)

	// Then: Should return config with hash
	require.NoError(t, err)
	require.NotNil(t, result, "Should return remote config")
	assert.NotNil(t, result.GetConfig(), "Should include config body")
	assert.NotNil(t, result.GetConfigHash(), "Should include config hash")
	assert.NotEmpty(t, result.GetConfigHash(), "Config hash should not be empty")

	// Then: Config body should match
	configFile := result.GetConfig().GetConfigMap()["opampcommander"]
	require.NotNil(t, configFile)
	assert.Equal(t, []byte(testConfig), configFile.GetBody())
}

func TestBuildRemoteConfig_ConfigAlreadyApplied(t *testing.T) {
	t.Parallel()

	// Given: Agent with config already applied
	testConfig := "receivers:\n  otlp:\nexporters:\n  logging:\n"

	mockUsecase := &mockAgentGroupUsecase{
		groups: []*agentgroup.AgentGroup{
			{
				Name: "test-group",
				AgentConfig: &agentgroup.AgentConfig{
					Value: testConfig,
				},
			},
		},
	}

	logger := slog.Default()
	service := &Service{
		logger:            logger,
		agentGroupUsecase: mockUsecase,
	}

	agentModel := model.NewAgent(uuid.New())
	agentModel.Metadata.Capabilities = agent.Capabilities(agent.AgentCapabilityAcceptsRemoteConfig)

	// Simulate that agent already received and applied this config
	// The hash should match what the server calculates
	configBytes := []byte(testConfig)
	configHash, err := vo.NewHash(configBytes)
	require.NoError(t, err)

	// Create a config data with the same hash that server would send
	configData := model.RemoteConfigData{
		Key:           configHash,
		Status:        model.RemoteConfigStatusUnset,
		Config:        configBytes,
		LastUpdatedAt: time.Now(),
	}

	// Apply the config data
	err = agentModel.Spec.RemoteConfig.ApplyRemoteConfig(configData)
	require.NoError(t, err)

	// Agent reports back that it applied the config
	agentModel.Spec.RemoteConfig.SetStatus(configHash, model.RemoteConfigStatusApplied)

	// Verify status was set
	currentStatus := agentModel.Spec.RemoteConfig.GetStatus(configHash)
	require.Equal(t, model.RemoteConfigStatusApplied, currentStatus, "Status should be set to Applied")

	// When: Build remote config
	result, err := service.buildRemoteConfig(context.Background(), agentModel)

	// Then: Should return hash only, no config body
	require.NoError(t, err)
	require.NotNil(t, result, "Should return remote config")
	assert.Nil(t, result.GetConfig(), "Should NOT include config body when already applied")
	assert.NotNil(t, result.GetConfigHash(), "Should include config hash")
	assert.NotEmpty(t, result.GetConfigHash(), "Config hash should not be empty")
	assert.Equal(t, configHash.Bytes(), result.GetConfigHash(), "Hash should match")
}

func TestBuildRemoteConfig_EmptyConfigValue(t *testing.T) {
	t.Parallel()

	// Given: Group with empty config value
	mockUsecase := &mockAgentGroupUsecase{
		groups: []*agentgroup.AgentGroup{
			{
				Name:        "test-group",
				AgentConfig: &agentgroup.AgentConfig{Value: ""},
			},
		},
	}

	logger := slog.Default()
	service := &Service{
		logger:            logger,
		agentGroupUsecase: mockUsecase,
	}

	agentModel := model.NewAgent(uuid.New())
	agentModel.Metadata.Capabilities = agent.Capabilities(agent.AgentCapabilityAcceptsRemoteConfig)

	// When: Build remote config
	result, err := service.buildRemoteConfig(context.Background(), agentModel)

	// Then: Should return nil
	require.NoError(t, err)
	assert.Nil(t, result, "Should not send config when value is empty")
}
