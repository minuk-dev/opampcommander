package auth_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/api/v1/auth"
)

func TestDeviceAuthnTokenResponse_TimeSerialization(t *testing.T) {
	expiryTime := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)

	response := auth.DeviceAuthnTokenResponse{
		DeviceCode:              "device123",
		UserCode:                "USER-CODE",
		VerificationURI:         "https://example.com/device",
		VerificationURIComplete: "https://example.com/device?code=USER-CODE",
		Expiry:                  v1.NewTime(expiryTime),
		Interval:                5,
	}

	// Test JSON marshaling
	data, err := json.Marshal(response)
	require.NoError(t, err)
	assert.Contains(t, string(data), "2024-01-15T10:30:45Z")

	// Test JSON unmarshaling
	var unmarshaledResponse auth.DeviceAuthnTokenResponse

	err = json.Unmarshal(data, &unmarshaledResponse)
	require.NoError(t, err)
	assert.Equal(t, response.DeviceCode, unmarshaledResponse.DeviceCode)
	assert.True(t, response.Expiry.Equal(&unmarshaledResponse.Expiry))
}

func TestDeviceAuthnTokenResponse_ZeroTimeSerialization(t *testing.T) {
	response := auth.DeviceAuthnTokenResponse{
		DeviceCode:      "device123",
		UserCode:        "USER-CODE",
		VerificationURI: "https://example.com/device",
		Expiry:          v1.Time{},
		Interval:        5,
	}

	// Test JSON marshaling - zero time should serialize as null
	data, err := json.Marshal(response)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"expiry":null`)
}
