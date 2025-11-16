package model_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/agent"
)

func TestNewAgent(t *testing.T) {
	t.Parallel()
	t.Run("Create agent with default values", func(t *testing.T) {
		t.Parallel()

		instanceUID := uuid.New()
		a := model.NewAgent(instanceUID)

		assert.Equal(t, instanceUID, a.Metadata.InstanceUID)
		assert.NotNil(t, a.Spec.RemoteConfig)
		assert.NotNil(t, a.Status.EffectiveConfig.ConfigMap.ConfigMap)
		assert.NotNil(t, a.Status.PackageStatuses.Packages)
		assert.NotNil(t, a.Status.ComponentHealth.ComponentHealthMap)
		assert.NotNil(t, a.Status.AvailableComponents.Components)
		assert.NotNil(t, a.Commands.Commands)
	})

	t.Run("Create agent with description option", func(t *testing.T) {
		t.Parallel()

		instanceUID := uuid.New()
		description := &agent.Description{
			IdentifyingAttributes: map[string]string{
				"service.name": "test-service",
			},
			NonIdentifyingAttributes: map[string]string{
				"os.type": "linux",
			},
		}

		a := model.NewAgent(instanceUID, model.WithDescription(description))

		assert.Equal(t, instanceUID, a.Metadata.InstanceUID)
		assert.Equal(t, "test-service", a.Metadata.Description.IdentifyingAttributes["service.name"])
		assert.Equal(t, "linux", a.Metadata.Description.NonIdentifyingAttributes["os.type"])
	})

	t.Run("Create agent with capabilities option", func(t *testing.T) {
		t.Parallel()

		instanceUID := uuid.New()
		capabilities := agent.Capabilities(agent.AgentCapabilityReportsStatus | agent.AgentCapabilityAcceptsRemoteConfig)

		a := model.NewAgent(instanceUID, model.WithCapabilities(&capabilities))

		assert.Equal(t, instanceUID, a.Metadata.InstanceUID)
		assert.Equal(t, capabilities, a.Metadata.Capabilities)
	})

	t.Run("Create agent with multiple options", func(t *testing.T) {
		t.Parallel()

		instanceUID := uuid.New()
		description := &agent.Description{
			IdentifyingAttributes: map[string]string{
				"service.name": "test-service",
			},
		}
		capabilities := agent.Capabilities(agent.AgentCapabilityReportsStatus)
		customCaps := &model.AgentCustomCapabilities{
			Capabilities: []string{"custom1", "custom2"},
		}

		a := model.NewAgent(
			instanceUID,
			model.WithDescription(description),
			model.WithCapabilities(&capabilities),
			model.WithCustomCapabilities(customCaps),
		)

		assert.Equal(t, instanceUID, a.Metadata.InstanceUID)
		assert.Equal(t, "test-service", a.Metadata.Description.IdentifyingAttributes["service.name"])
		assert.Equal(t, capabilities, a.Metadata.Capabilities)
		assert.Equal(t, []string{"custom1", "custom2"}, a.Metadata.CustomCapabilities.Capabilities)
	})
}

func TestAgentMetadata_IsComplete(t *testing.T) {
	t.Parallel()
	t.Run("Empty metadata is not complete", func(t *testing.T) {
		t.Parallel()

		metadata := model.AgentMetadata{
			InstanceUID: uuid.New(),
		}

		assert.False(t, metadata.IsComplete())
	})

	t.Run("Metadata with only description is not complete", func(t *testing.T) {
		t.Parallel()

		metadata := model.AgentMetadata{
			InstanceUID: uuid.New(),
			Description: agent.Description{
				IdentifyingAttributes: map[string]string{
					"service.name": "test",
				},
			},
		}

		assert.False(t, metadata.IsComplete())
	})

	t.Run("Metadata with only capabilities is not complete", func(t *testing.T) {
		t.Parallel()

		metadata := model.AgentMetadata{
			InstanceUID:  uuid.New(),
			Capabilities: agent.Capabilities(agent.AgentCapabilityReportsStatus),
		}

		assert.False(t, metadata.IsComplete())
	})

	t.Run("Metadata with description and capabilities is complete", func(t *testing.T) {
		t.Parallel()

		metadata := model.AgentMetadata{
			InstanceUID: uuid.New(),
			Description: agent.Description{
				IdentifyingAttributes: map[string]string{
					"service.name": "test",
				},
			},
			Capabilities: agent.Capabilities(agent.AgentCapabilityReportsStatus),
		}

		assert.True(t, metadata.IsComplete())
	})

	t.Run("Metadata with non-identifying attributes and capabilities is complete", func(t *testing.T) {
		t.Parallel()

		metadata := model.AgentMetadata{
			InstanceUID: uuid.New(),
			Description: agent.Description{
				NonIdentifyingAttributes: map[string]string{
					"os.type": "linux",
				},
			},
			Capabilities: agent.Capabilities(agent.AgentCapabilityReportsStatus),
		}

		assert.True(t, metadata.IsComplete())
	})
}

func TestAgentCommands_HasReportFullStateCommand(t *testing.T) {
	t.Parallel()
	t.Run("Empty commands returns false", func(t *testing.T) {
		t.Parallel()

		commands := model.AgentCommands{
			Commands: []model.AgentCommand{},
		}

		assert.False(t, commands.HasReportFullStateCommand())
	})

	t.Run("Commands with ReportFullState returns true", func(t *testing.T) {
		t.Parallel()

		commands := model.AgentCommands{
			Commands: []model.AgentCommand{
				{
					CommandID:       uuid.New(),
					ReportFullState: true,
					CreatedAt:       time.Now(),
					CreatedBy:       "test",
				},
			},
		}

		assert.True(t, commands.HasReportFullStateCommand())
	})

	t.Run("Commands without ReportFullState returns false", func(t *testing.T) {
		t.Parallel()

		commands := model.AgentCommands{
			Commands: []model.AgentCommand{
				{
					CommandID:       uuid.New(),
					ReportFullState: false,
					CreatedAt:       time.Now(),
					CreatedBy:       "test",
				},
			},
		}

		assert.False(t, commands.HasReportFullStateCommand())
	})
}

func TestAgent_UpdateLastCommunicationInfo(t *testing.T) {
	t.Parallel()
	t.Run("Update last communication info with connection", func(t *testing.T) {
		t.Parallel()

		a := model.NewAgent(uuid.New())
		connection := model.NewConnection("conn-id", model.ConnectionTypeWebSocket)
		now := time.Now()

		a.UpdateLastCommunicationInfo(now, connection)
		assert.Equal(t, now, a.Status.LastReportedAt)
		assert.Equal(t, model.ConnectionTypeWebSocket, a.Status.ConnectionType)
	})

	t.Run("Update last communication info without connection", func(t *testing.T) {
		t.Parallel()

		a := model.NewAgent(uuid.New())
		now := time.Now()

		a.UpdateLastCommunicationInfo(now, nil)
		assert.Equal(t, now, a.Status.LastReportedAt)
		assert.Equal(t, model.ConnectionTypeUnknown, a.Status.ConnectionType)
	})
}
