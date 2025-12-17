package integration_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/internal/application/service/agent"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/model/agent"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

func TestRestartAgentIntegration(t *testing.T) {
	t.Parallel()

	t.Run("restart agent with restart capability", func(t *testing.T) {
		t.Parallel()

		// Setup
		ctx := context.Background()

		// Create mock usecases
		agentUsecase := &mockAgentUsecase{
			agents: make(map[uuid.UUID]*model.Agent),
		}
		agentNotificationUsecase := &mockAgentNotificationUsecase{}

		// Create agent service
		agentService := agent.New(
			agentUsecase,
			agentNotificationUsecase,
			slog.Default(),
		)

		// Create test agent with restart capability
		instanceUID := uuid.New()
		capabilities := agentmodel.Capabilities(agentmodel.AgentCapabilityAcceptsRestartCommand)
		testAgent := model.NewAgent(instanceUID,
			model.WithCapabilities(&capabilities),
			model.WithComponentHealth(&model.AgentComponentHealth{
				StartTime: time.Now().Add(-1 * time.Hour), // Started 1 hour ago
				Healthy:   true,
			}),
		)

		agentUsecase.agents[instanceUID] = testAgent

		// Execute restart
		err := agentService.RestartAgent(ctx, instanceUID)
		require.NoError(t, err)

		// Verify agent was updated with restart flag
		updatedAgent := agentUsecase.agents[instanceUID]
		assert.True(t, updatedAgent.ShouldBeRestarted(), "Agent should be flagged for restart")
		assert.False(t, updatedAgent.Spec.RequiredRestartedAt.IsZero(), "RequiredRestartedAt should be set")
		assert.True(t, updatedAgent.Spec.RequiredRestartedAt.After(testAgent.Status.ComponentHealth.StartTime),
			"RequiredRestartedAt should be after original StartTime")

		// Verify notification was called
		assert.True(t, agentNotificationUsecase.notificationCalled, "Agent notification should have been called")
	})

	t.Run("restart agent without restart capability should fail", func(t *testing.T) {
		t.Parallel()

		// Setup
		ctx := context.Background()

		// Create mock usecases
		agentUsecase := &mockAgentUsecase{
			agents: make(map[uuid.UUID]*model.Agent),
		}
		agentNotificationUsecase := &mockAgentNotificationUsecase{}

		// Create agent service
		agentService := agent.New(
			agentUsecase,
			agentNotificationUsecase,
			slog.Default(),
		)

		// Create test agent WITHOUT restart capability
		instanceUID := uuid.New()
		capabilities := agentmodel.Capabilities(agentmodel.AgentCapabilityReportsStatus) // Only status capability
		testAgent := model.NewAgent(instanceUID,
			model.WithCapabilities(&capabilities),
		)

		agentUsecase.agents[instanceUID] = testAgent

		// Execute restart - should fail
		err := agentService.RestartAgent(ctx, instanceUID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not support restart capability")

		// Verify agent was not updated
		updatedAgent := agentUsecase.agents[instanceUID]
		assert.False(t, updatedAgent.ShouldBeRestarted(), "Agent should not be flagged for restart")
		assert.True(t, updatedAgent.Spec.RequiredRestartedAt.IsZero(), "RequiredRestartedAt should remain zero")
	})
}

// Mock implementations.
type mockAgentUsecase struct {
	agents map[uuid.UUID]*model.Agent
}

func (m *mockAgentUsecase) GetAgent(_ context.Context, instanceUID uuid.UUID) (*model.Agent, error) {
	agent, exists := m.agents[instanceUID]
	if !exists {
		return nil, port.ErrResourceNotExist
	}

	return agent, nil
}

func (m *mockAgentUsecase) GetOrCreateAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error) {
	return m.GetAgent(ctx, instanceUID)
}

var errNotImplemented = errors.New("not implemented")

func (m *mockAgentUsecase) ListAgentsBySelector(
	_ context.Context,
	_ model.AgentSelector,
	_ *model.ListOptions,
) (*model.ListResponse[*model.Agent], error) {
	return nil, errNotImplemented
}

func (m *mockAgentUsecase) SaveAgent(_ context.Context, agent *model.Agent) error {
	m.agents[agent.Metadata.InstanceUID] = agent

	return nil
}

func (m *mockAgentUsecase) ListAgents(
	_ context.Context,
	_ *model.ListOptions,
) (*model.ListResponse[*model.Agent], error) {
	return nil, errNotImplemented
}

type mockAgentNotificationUsecase struct {
	notificationCalled bool
}

func (m *mockAgentNotificationUsecase) NotifyAgentUpdated(_ context.Context, _ *model.Agent) error {
	m.notificationCalled = true

	return nil
}

func (m *mockAgentNotificationUsecase) RestartAgent(_ context.Context, _ uuid.UUID) error {
	return nil
}
