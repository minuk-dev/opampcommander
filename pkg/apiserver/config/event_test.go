package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
)

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
