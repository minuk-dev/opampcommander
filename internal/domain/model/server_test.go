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
			Conditions:      []model.Condition{},
		}

		// Set registered condition
		server.MarkRegistered("system")

		condition := server.GetCondition(model.ConditionTypeCreated)
		assert.NotNil(t, condition)
		assert.Equal(t, model.ConditionTypeCreated, condition.Type)
		assert.Equal(t, model.ConditionStatusTrue, condition.Status)
		assert.Equal(t, "system", condition.Reason)
		assert.Equal(t, "Server registered", condition.Message)
		assert.True(t, server.IsConditionTrue(model.ConditionTypeCreated))
	})

	t.Run("Mark server as alive", func(t *testing.T) {
		t.Parallel()

		server := &model.Server{
			ID:              "test-server",
			LastHeartbeatAt: time.Now(),
			Conditions:      []model.Condition{},
		}

		server.SetCondition(model.ConditionTypeAlive, model.ConditionStatusTrue, "heartbeat", "Server is alive")

		assert.True(t, server.IsConditionTrue(model.ConditionTypeAlive))

		condition := server.GetCondition(model.ConditionTypeAlive)
		assert.NotNil(t, condition)
		assert.Equal(t, model.ConditionTypeAlive, condition.Type)
		assert.Equal(t, model.ConditionStatusTrue, condition.Status)
		assert.Equal(t, "heartbeat", condition.Reason)
		assert.Equal(t, "Server is alive", condition.Message)
	})

	t.Run("Mark server as not alive", func(t *testing.T) {
		t.Parallel()

		server := &model.Server{
			ID:              "test-server",
			LastHeartbeatAt: time.Now(),
			Conditions:      []model.Condition{},
		}

		// First mark as alive
		server.SetCondition(model.ConditionTypeAlive, model.ConditionStatusTrue, "heartbeat", "Server is alive")
		assert.True(t, server.IsConditionTrue(model.ConditionTypeAlive))

		// Then mark as not alive
		server.SetCondition(model.ConditionTypeAlive, model.ConditionStatusFalse, "timeout", "Server is not responding")

		assert.False(t, server.IsConditionTrue(model.ConditionTypeAlive))

		condition := server.GetCondition(model.ConditionTypeAlive)
		assert.NotNil(t, condition)
		assert.Equal(t, model.ConditionTypeAlive, condition.Type)
		assert.Equal(t, model.ConditionStatusFalse, condition.Status)
		assert.Equal(t, "timeout", condition.Reason)
		assert.Equal(t, "Server is not responding", condition.Message)
	})

	t.Run("Get registered at and by", func(t *testing.T) {
		t.Parallel()

		server := &model.Server{
			ID:              "test-server",
			LastHeartbeatAt: time.Now(),
			Conditions:      []model.Condition{},
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
			Conditions:      []model.Condition{},
		}

		condition := server.GetCondition(model.ConditionTypeAlive)
		assert.Nil(t, condition)
		assert.False(t, server.IsConditionTrue(model.ConditionTypeAlive))
	})

	t.Run("Update existing condition", func(t *testing.T) {
		t.Parallel()

		server := &model.Server{
			ID:              "test-server",
			LastHeartbeatAt: time.Now(),
			Conditions:      []model.Condition{},
		}

		// First mark as alive
		server.SetCondition(model.ConditionTypeAlive, model.ConditionStatusTrue, "initial", "Server is alive")
		assert.True(t, server.IsConditionTrue(model.ConditionTypeAlive))

		// Get the initial timestamp
		firstCondition := server.GetCondition(model.ConditionTypeAlive)
		firstTime := firstCondition.LastTransitionTime

		// Wait a moment and mark as not alive
		time.Sleep(time.Millisecond)
		server.SetCondition(model.ConditionTypeAlive, model.ConditionStatusFalse, "timeout", "Server is not responding")

		// Should update the existing condition
		assert.Len(t, server.Conditions, 1) // Still only one condition
		assert.False(t, server.IsConditionTrue(model.ConditionTypeAlive))

		updatedCondition := server.GetCondition(model.ConditionTypeAlive)
		assert.True(t, updatedCondition.LastTransitionTime.After(firstTime))
		assert.Equal(t, "timeout", updatedCondition.Reason)
		assert.Equal(t, model.ConditionStatusFalse, updatedCondition.Status)
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
			Conditions:      []model.Condition{},
		}

		assert.True(t, server.IsAlive(now, 1*time.Minute))
	})

	t.Run("Server is not alive beyond timeout", func(t *testing.T) {
		t.Parallel()

		now := time.Now()
		server := &model.Server{
			ID:              "test-server",
			LastHeartbeatAt: now.Add(-2 * time.Minute),
			Conditions:      []model.Condition{},
		}

		assert.False(t, server.IsAlive(now, 1*time.Minute))
	})
}
