package connection_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/api/v1/connection"
)

func TestConnection_TimeSerialization(t *testing.T) {
	t.Parallel()

	connID := uuid.New()
	instanceID := uuid.New()
	lastComm := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)

	conn := connection.Connection{
		ID:                 connID,
		InstanceUID:        instanceID,
		Type:               "websocket",
		LastCommunicatedAt: v1.NewTime(lastComm),
		Alive:              true,
	}

	// Test JSON marshaling
	data, err := json.Marshal(conn)
	require.NoError(t, err)
	assert.Contains(t, string(data), "2024-01-15T10:30:45Z")

	// Test JSON unmarshaling
	var unmarshaledConn connection.Connection

	err = json.Unmarshal(data, &unmarshaledConn)
	require.NoError(t, err)
	assert.Equal(t, conn.ID, unmarshaledConn.ID)
	assert.True(t, conn.LastCommunicatedAt.Equal(&unmarshaledConn.LastCommunicatedAt))
	assert.Equal(t, conn.Alive, unmarshaledConn.Alive)
}

func TestConnection_ZeroTimeSerialization(t *testing.T) {
	t.Parallel()

	connID := uuid.New()
	instanceID := uuid.New()

	conn := connection.Connection{
		ID:                 connID,
		InstanceUID:        instanceID,
		Type:               "websocket",
		LastCommunicatedAt: v1.Time{},
		Alive:              false,
	}

	// Test JSON marshaling - zero time should serialize as null
	data, err := json.Marshal(conn)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"lastCommunicatedAt":null`)
}
