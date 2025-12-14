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
)

// Mock AgentGroupUsecase for testing.
type mockAgentGroupUsecase struct {
	groups []*model.AgentGroup
}

func (m *mockAgentGroupUsecase) GetAgentGroup(_ context.Context, _ string) (*model.AgentGroup, error) {
	panic("not implemented")
}

func (m *mockAgentGroupUsecase) ListAgentGroups(
	_ context.Context,
	_ *model.ListOptions,
) (*model.ListResponse[*model.AgentGroup], error) {
	panic("not implemented")
}

func (m *mockAgentGroupUsecase) SaveAgentGroup(
	_ context.Context,
	_ string,
	agentGroup *model.AgentGroup,
) (*model.AgentGroup, error) {
	return agentGroup, nil
}

func (m *mockAgentGroupUsecase) DeleteAgentGroup(_ context.Context, _ string, _ time.Time, _ string) error {
	panic("not implemented")
}

func (m *mockAgentGroupUsecase) GetAgentGroupsForAgent(
	_ context.Context,
	_ *model.Agent,
) ([]*model.AgentGroup, error) {
	return m.groups, nil
}

func TestFetchServerToAgent_NoRemoteConfigCapability(t *testing.T) {
	t.Parallel()

	// Given: Agent without AcceptsRemoteConfig capability
	logger := slog.Default()
	service := &Service{
		logger:            logger,
		agentGroupUsecase: &mockAgentGroupUsecase{},
	}

	agentModel := model.NewAgent(uuid.New())
	agentModel.Metadata.Capabilities = agent.Capabilities(0) // No capabilities

	// When: Fetch ServerToAgent message
	result, err := service.fetchServerToAgent(context.Background(), agentModel)

	// Then: Should return message without remote config
	require.NoError(t, err)
	assert.NotNil(t, result, "Result should not be nil")
	assert.Nil(t, result.RemoteConfig, "RemoteConfig should be nil when agent has no capability")
}

func TestFetchServerToAgent_WithRemoteConfig(t *testing.T) {
	t.Parallel()

	// Given: Agent with AcceptsRemoteConfig capability and config
	logger := slog.Default()
	service := &Service{
		logger:            logger,
		agentGroupUsecase: &mockAgentGroupUsecase{},
	}

	agentModel := model.NewAgent(uuid.New())
	agentModel.Metadata.Capabilities = agent.Capabilities(agent.AgentCapabilityAcceptsRemoteConfig)

	testConfig := []byte("test config")
	err := agentModel.Spec.RemoteConfig.ApplyRemoteConfig(testConfig, "application/yaml")
	require.NoError(t, err)

	// When: Fetch ServerToAgent message
	result, err := service.fetchServerToAgent(context.Background(), agentModel)

	// Then: Should return message with remote config
	require.NoError(t, err)
	assert.NotNil(t, result, "Result should not be nil")
	assert.NotNil(t, result.RemoteConfig, "RemoteConfig should be present")
	assert.Equal(t, testConfig, result.RemoteConfig.Config.ConfigMap["opampcommander"].Body)
	assert.NotEmpty(t, result.RemoteConfig.ConfigHash, "ConfigHash should be present")
}