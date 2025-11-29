package model_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

func TestServerConditions(t *testing.T) {
	t.Parallel()

	t.Run("Set and get condition", func(t *testing.T) {
		t.Parallel()

		server := &model.Server{
			ID:              "test-server",
			LastHeartbeatAt: time.Now(),
			Conditions:      []model.ServerCondition{},
		}

		// Set registered condition
		server.MarkRegistered("system")

		condition := server.GetCondition(model.ServerConditionTypeRegistered)
		assert.NotNil(t, condition)
		assert.Equal(t, model.ServerConditionTypeRegistered, condition.Type)
		assert.Equal(t, model.ServerConditionStatusTrue, condition.Status)
		assert.Equal(t, "system", condition.Reason)
		assert.Equal(t, "Server registered", condition.Message)
		assert.True(t, server.IsConditionTrue(model.ServerConditionTypeRegistered))
	})

	t.Run("Mark server as alive", func(t *testing.T) {
		t.Parallel()

		server := &model.Server{
			ID:              "test-server",
			LastHeartbeatAt: time.Now(),
			Conditions:      []model.ServerCondition{},
		}

		server.MarkAlive("heartbeat")

		assert.True(t, server.IsConditionTrue(model.ServerConditionTypeAlive))

		condition := server.GetCondition(model.ServerConditionTypeAlive)
		assert.NotNil(t, condition)
		assert.Equal(t, model.ServerConditionTypeAlive, condition.Type)
		assert.Equal(t, model.ServerConditionStatusTrue, condition.Status)
		assert.Equal(t, "heartbeat", condition.Reason)
		assert.Equal(t, "Server is alive", condition.Message)
	})

	t.Run("Mark server as not alive", func(t *testing.T) {
		t.Parallel()

		server := &model.Server{
			ID:              "test-server",
			LastHeartbeatAt: time.Now(),
			Conditions:      []model.ServerCondition{},
		}

		// First mark as alive
		server.MarkAlive("heartbeat")
		assert.True(t, server.IsConditionTrue(model.ServerConditionTypeAlive))

		// Then mark as not alive
		server.MarkNotAlive("timeout")

		assert.False(t, server.IsConditionTrue(model.ServerConditionTypeAlive))

		condition := server.GetCondition(model.ServerConditionTypeAlive)
		assert.NotNil(t, condition)
		assert.Equal(t, model.ServerConditionTypeAlive, condition.Type)
		assert.Equal(t, model.ServerConditionStatusFalse, condition.Status)
		assert.Equal(t, "timeout", condition.Reason)
		assert.Equal(t, "Server is not responding", condition.Message)
	})

	t.Run("Get registered at and by", func(t *testing.T) {
		t.Parallel()

		server := &model.Server{
			ID:              "test-server",
			LastHeartbeatAt: time.Now(),
			Conditions:      []model.ServerCondition{},
		}

		// Initially should return nil/empty
		assert.Nil(t, server.GetRegisteredAt())
		assert.Empty(t, server.GetRegisteredBy())

		// Mark as registered
		server.MarkRegistered("admin")

		registeredAt := server.GetRegisteredAt()
		assert.NotNil(t, registeredAt)
		assert.Equal(t, "admin", server.GetRegisteredBy())
	})

	t.Run("Get non-existent condition", func(t *testing.T) {
		t.Parallel()

		server := &model.Server{
			ID:              "test-server",
			LastHeartbeatAt: time.Now(),
			Conditions:      []model.ServerCondition{},
		}

		condition := server.GetCondition(model.ServerConditionTypeAlive)
		assert.Nil(t, condition)
		assert.False(t, server.IsConditionTrue(model.ServerConditionTypeAlive))
	})

	t.Run("Update existing condition", func(t *testing.T) {
		t.Parallel()

		server := &model.Server{
			ID:              "test-server",
			LastHeartbeatAt: time.Now(),
			Conditions:      []model.ServerCondition{},
		}

		// First mark as alive
		server.MarkAlive("initial")
		assert.True(t, server.IsConditionTrue(model.ServerConditionTypeAlive))

		// Get the initial timestamp
		firstCondition := server.GetCondition(model.ServerConditionTypeAlive)
		firstTime := firstCondition.LastTransitionTime

		// Wait a moment and mark as not alive
		time.Sleep(time.Millisecond)
		server.MarkNotAlive("timeout")

		// Should update the existing condition
		assert.Len(t, server.Conditions, 1) // Still only one condition
		assert.False(t, server.IsConditionTrue(model.ServerConditionTypeAlive))

		updatedCondition := server.GetCondition(model.ServerConditionTypeAlive)
		assert.True(t, updatedCondition.LastTransitionTime.After(firstTime))
		assert.Equal(t, "timeout", updatedCondition.Reason)
		assert.Equal(t, model.ServerConditionStatusFalse, updatedCondition.Status)
	})
}

func TestServerIsAlive(t *testing.T) {
	t.Parallel()

	t.Run("Server is alive within timeout", func(t *testing.T) {
		t.Parallel()

		now := time.Now()
		server := &model.Server{
			ID:              "test-server",
			LastHeartbeatAt: now.Add(-30 * time.Second),
			Conditions:      []model.ServerCondition{},
		}

		assert.True(t, server.IsAlive(now, 1*time.Minute))
	})

	t.Run("Server is not alive beyond timeout", func(t *testing.T) {
		t.Parallel()

		now := time.Now()
		server := &model.Server{
			ID:              "test-server",
			LastHeartbeatAt: now.Add(-2 * time.Minute),
			Conditions:      []model.ServerCondition{},
		}

		assert.False(t, server.IsAlive(now, 1*time.Minute))
	})
}
