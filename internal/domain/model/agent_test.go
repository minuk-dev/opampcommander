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
		assert.Equal(t, uint64(0), a.Status.SequenceNum)
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

func TestAgent_RecordLastReported(t *testing.T) {
	t.Parallel()
	t.Run("Record last reported with server and sequence number", func(t *testing.T) {
		t.Parallel()

		a := model.NewAgent(uuid.New())
		server := &model.Server{
			ID: "test-server",
		}
		now := time.Now()
		sequenceNum := uint64(123)

		a.RecordLastReported(server, now, sequenceNum)

		assert.Equal(t, server.ID, a.Status.LastReportedTo)
		assert.Equal(t, now, a.Status.LastReportedAt)
		assert.Equal(t, sequenceNum, a.Status.SequenceNum)
	})

	t.Run("Record last reported without server", func(t *testing.T) {
		t.Parallel()

		a := model.NewAgent(uuid.New())
		now := time.Now()
		sequenceNum := uint64(456)

		a.RecordLastReported(nil, now, sequenceNum)

		assert.Empty(t, a.Status.LastReportedTo)
		assert.Equal(t, now, a.Status.LastReportedAt)
		assert.Equal(t, sequenceNum, a.Status.SequenceNum)
	})

	t.Run("Record last reported with incremental sequence numbers", func(t *testing.T) {
		t.Parallel()

		a := model.NewAgent(uuid.New())
		server := &model.Server{
			ID: "test-server",
		}
		now := time.Now()

		// First report
		a.RecordLastReported(server, now, 1)
		assert.Equal(t, uint64(1), a.Status.SequenceNum)

		// Second report
		a.RecordLastReported(server, now.Add(time.Second), 2)
		assert.Equal(t, uint64(2), a.Status.SequenceNum)

		// Third report
		a.RecordLastReported(server, now.Add(2*time.Second), 3)
		assert.Equal(t, uint64(3), a.Status.SequenceNum)
	})

	t.Run("Record last reported with zero sequence number", func(t *testing.T) {
		t.Parallel()

		a := model.NewAgent(uuid.New())
		server := &model.Server{
			ID: "test-server",
		}
		now := time.Now()

		a.RecordLastReported(server, now, 0)

		assert.Equal(t, server.ID, a.Status.LastReportedTo)
		assert.Equal(t, now, a.Status.LastReportedAt)
		assert.Equal(t, uint64(0), a.Status.SequenceNum)
	})

	t.Run("Record last reported updates existing values", func(t *testing.T) {
		t.Parallel()

		a := model.NewAgent(uuid.New())
		server1 := &model.Server{
			ID: "server-1",
		}
		server2 := &model.Server{
			ID: "server-2",
		}
		time1 := time.Now()
		time2 := time1.Add(time.Hour)

		// First report
		a.RecordLastReported(server1, time1, 100)
		assert.Equal(t, server1.ID, a.Status.LastReportedTo)
		assert.Equal(t, time1, a.Status.LastReportedAt)
		assert.Equal(t, uint64(100), a.Status.SequenceNum)

		// Second report with different server
		a.RecordLastReported(server2, time2, 200)
		assert.Equal(t, server2.ID, a.Status.LastReportedTo)
		assert.Equal(t, time2, a.Status.LastReportedAt)
		assert.Equal(t, uint64(200), a.Status.SequenceNum)
	})
}

func TestAgentConditions(t *testing.T) {
	t.Parallel()

	t.Run("New agent should have registered condition", func(t *testing.T) {
		t.Parallel()

		agent := model.NewAgent(uuid.New())

		// Check that the agent has the registered condition
		condition := agent.GetCondition(model.AgentConditionTypeRegistered)
		assert.NotNil(t, condition)
		assert.Equal(t, model.AgentConditionTypeRegistered, condition.Type)
		assert.Equal(t, model.AgentConditionStatusTrue, condition.Status)
		assert.Equal(t, "system", condition.Reason)
		assert.Equal(t, "Agent registered", condition.Message)
		assert.True(t, agent.IsConditionTrue(model.AgentConditionTypeRegistered))
	})

	t.Run("Mark agent as connected", func(t *testing.T) {
		t.Parallel()

		agent := model.NewAgent(uuid.New())
		triggeredBy := "user"

		agent.MarkConnected(triggeredBy)

		assert.True(t, agent.Status.Connected)
		assert.True(t, agent.IsConditionTrue(model.AgentConditionTypeConnected))

		condition := agent.GetCondition(model.AgentConditionTypeConnected)
		assert.NotNil(t, condition)
		assert.Equal(t, model.AgentConditionTypeConnected, condition.Type)
		assert.Equal(t, model.AgentConditionStatusTrue, condition.Status)
		assert.Equal(t, triggeredBy, condition.Reason)
		assert.Equal(t, "Agent connected", condition.Message)
	})

	t.Run("Mark agent as disconnected", func(t *testing.T) {
		t.Parallel()

		agent := model.NewAgent(uuid.New())
		triggeredBy := "system"

		// First connect
		agent.MarkConnected("user")
		assert.True(t, agent.Status.Connected)

		// Then disconnect
		agent.MarkDisconnected(triggeredBy)

		assert.False(t, agent.Status.Connected)
		assert.False(t, agent.IsConditionTrue(model.AgentConditionTypeConnected))

		condition := agent.GetCondition(model.AgentConditionTypeConnected)
		assert.NotNil(t, condition)
		assert.Equal(t, model.AgentConditionTypeConnected, condition.Type)
		assert.Equal(t, model.AgentConditionStatusFalse, condition.Status)
		assert.Equal(t, triggeredBy, condition.Reason)
		assert.Equal(t, "Agent disconnected", condition.Message)
	})

	t.Run("Mark agent as healthy", func(t *testing.T) {
		t.Parallel()

		agent := model.NewAgent(uuid.New())
		triggeredBy := "health-check"

		agent.MarkHealthy(triggeredBy)

		assert.True(t, agent.IsConditionTrue(model.AgentConditionTypeHealthy))

		condition := agent.GetCondition(model.AgentConditionTypeHealthy)
		assert.NotNil(t, condition)
		assert.Equal(t, model.AgentConditionTypeHealthy, condition.Type)
		assert.Equal(t, model.AgentConditionStatusTrue, condition.Status)
		assert.Equal(t, triggeredBy, condition.Reason)
		assert.Equal(t, "Agent is healthy", condition.Message)
	})

	t.Run("Mark agent as unhealthy", func(t *testing.T) {
		t.Parallel()

		agent := model.NewAgent(uuid.New())
		triggeredBy := "health-check"
		reason := "high CPU usage"

		agent.MarkUnhealthy(triggeredBy, reason)

		assert.False(t, agent.IsConditionTrue(model.AgentConditionTypeHealthy))

		condition := agent.GetCondition(model.AgentConditionTypeHealthy)
		assert.NotNil(t, condition)
		assert.Equal(t, model.AgentConditionTypeHealthy, condition.Type)
		assert.Equal(t, model.AgentConditionStatusFalse, condition.Status)
		assert.Equal(t, triggeredBy, condition.Reason)
		assert.Contains(t, condition.Message, reason)
	})

	t.Run("Get non-existent condition", func(t *testing.T) {
		t.Parallel()

		agent := model.NewAgent(uuid.New())

		condition := agent.GetCondition(model.AgentConditionTypeHealthy)
		assert.Nil(t, condition)
		assert.False(t, agent.IsConditionTrue(model.AgentConditionTypeHealthy))
	})
}
