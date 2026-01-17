package server_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/api/v1/server"
)

func TestServer_TimeSerialization(t *testing.T) {
	t.Parallel()

	heartbeatTime := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)

	srv := server.Server{
		ID:              "server-1",
		LastHeartbeatAt: v1.NewTime(heartbeatTime),
		Conditions:      []server.Condition{},
	}

	// Test JSON marshaling
	data, err := json.Marshal(srv)
	require.NoError(t, err)
	assert.Contains(t, string(data), "2024-01-15T10:30:45Z")

	// Test JSON unmarshaling
	var unmarshaledServer server.Server

	err = json.Unmarshal(data, &unmarshaledServer)
	require.NoError(t, err)
	assert.Equal(t, srv.ID, unmarshaledServer.ID)
	assert.True(t, srv.LastHeartbeatAt.Equal(&unmarshaledServer.LastHeartbeatAt))
}

func TestServerCondition_TimeSerialization(t *testing.T) {
	t.Parallel()

	transitionTime := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)

	condition := server.Condition{
		Type:               server.ConditionTypeAlive,
		LastTransitionTime: v1.NewTime(transitionTime),
		Status:             server.ConditionStatusTrue,
		Reason:             "HeartbeatReceived",
		Message:            "Server is alive",
	}

	// Test JSON marshaling
	data, err := json.Marshal(condition)
	require.NoError(t, err)
	assert.Contains(t, string(data), "2024-01-15T10:30:45Z")

	// Test JSON unmarshaling
	var unmarshaledCondition server.Condition

	err = json.Unmarshal(data, &unmarshaledCondition)
	require.NoError(t, err)
	assert.Equal(t, condition.Type, unmarshaledCondition.Type)
	assert.True(t, condition.LastTransitionTime.Equal(&unmarshaledCondition.LastTransitionTime))
}

func TestServer_ZeroTimeSerialization(t *testing.T) {
	t.Parallel()

	srv := server.Server{
		ID:              "server-1",
		LastHeartbeatAt: v1.Time{},
		Conditions:      []server.Condition{},
	}

	// Test JSON marshaling - zero time should serialize as null
	data, err := json.Marshal(srv)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"lastHeartbeatAt":null`)
}
