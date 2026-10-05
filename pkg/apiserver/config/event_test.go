package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
)

func TestKafkaSettings_DefaultsAndOverrides(t *testing.T) {
	t.Parallel()

	settings := config.KafkaSettings{Brokers: []string{"broker:9092"}, Topic: "events"}.WithDefaults()
	require.NoError(t, settings.Validate())
	assert.Equal(t, 2*time.Second, settings.SendTimeout)
	assert.Equal(t, 100*time.Millisecond, settings.RetryBackoff)
	assert.Equal(t, 2, settings.RetryAttempts)
	assert.Equal(t, 2, settings.FailureThreshold)
	assert.Equal(t, 5*time.Second, settings.ProbeInterval)
	assert.Equal(t, []string{"broker:9092"}, settings.Brokers)
	assert.Equal(t, "events", settings.Topic)
	assert.Equal(t, config.DefaultKafkaSettings(), config.KafkaSettings{}.WithDefaults())

	overrides := config.KafkaSettings{
		SendTimeout: 3 * time.Second, RetryBackoff: 25 * time.Millisecond,
		RetryAttempts: 3, FailureThreshold: 4, ProbeInterval: 9 * time.Second,
	}
	assert.Equal(t, overrides, overrides.WithDefaults())
	require.NoError(t, overrides.Validate())
}

func TestKafkaSettings_RejectsNegativeValues(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name     string
		settings config.KafkaSettings
	}{
		{name: "sendTimeout", settings: config.KafkaSettings{SendTimeout: -time.Second}},
		{name: "retryBackoff", settings: config.KafkaSettings{RetryBackoff: -time.Second}},
		{name: "retryAttempts", settings: config.KafkaSettings{RetryAttempts: -1}},
		{name: "failureThreshold", settings: config.KafkaSettings{FailureThreshold: -1}},
		{name: "probeInterval", settings: config.KafkaSettings{ProbeInterval: -time.Second}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.ErrorIs(t, tt.settings.WithDefaults().Validate(), config.ErrKafkaSendSettingsInvalid)
		})
	}
}

func TestEventProtocolType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		protocol config.EventProtocolType
		want     string
	}{
		{"inmemory", config.EventProtocolTypeInMemory, "inmemory"},
		{"kafka", config.EventProtocolTypeKafka, "kafka"},
		{"direct", config.EventProtocolTypeDirect, "direct"},
		{"custom", config.EventProtocolType("custom"), "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.protocol.String())
		})
	}
}

func TestDirectSubProtocol_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		subProtocol config.DirectSubProtocol
		want        string
	}{
		{"http", config.DirectSubProtocolHTTP, "http"},
		{"grpc", config.DirectSubProtocolGRPC, "grpc"},
		{"custom", config.DirectSubProtocol("custom"), "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.subProtocol.String())
		})
	}
}
