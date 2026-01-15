package agentgroup_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/api/v1/agentgroup"
)

func TestCondition_TimeSerialization(t *testing.T) {
	testTime := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)

	condition := agentgroup.Condition{
		Type:               agentgroup.ConditionTypeCreated,
		LastTransitionTime: v1.NewTime(testTime),
		Status:             agentgroup.ConditionStatusTrue,
		Reason:             "TestReason",
		Message:            "Test message",
	}

	// Test JSON marshaling
	data, err := json.Marshal(condition)
	require.NoError(t, err)
	assert.Contains(t, string(data), "2024-01-15T10:30:45Z")

	// Test JSON unmarshaling
	var unmarshaledCondition agentgroup.Condition

	err = json.Unmarshal(data, &unmarshaledCondition)
	require.NoError(t, err)
	assert.Equal(t, condition.Type, unmarshaledCondition.Type)
	assert.True(t, condition.LastTransitionTime.Equal(&unmarshaledCondition.LastTransitionTime))
}

func TestCondition_ZeroTimeSerialization(t *testing.T) {
	condition := agentgroup.Condition{
		Type:               agentgroup.ConditionTypeCreated,
		LastTransitionTime: v1.Time{},
		Status:             agentgroup.ConditionStatusTrue,
		Reason:             "TestReason",
	}

	// Test JSON marshaling - zero time should serialize as null
	data, err := json.Marshal(condition)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"lastTransitionTime":null`)
}
