//nolint:testpackage // white-box test of the unexported protobuf->domain converters
package opamp

import (
	"testing"
	"time"

	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func strValue(s string) *protobufs.AnyValue {
	return &protobufs.AnyValue{Value: &protobufs.AnyValue_StringValue{StringValue: s}}
}

// TestDescToDomain_PreservesNonStringAttributes guards against the fidelity bug where only
// string AnyValues were kept: an agent reporting int/bool/double identifying attributes must
// have them preserved (as their string form), because AgentGroup selectors match on these.
func TestDescToDomain_PreservesNonStringAttributes(t *testing.T) {
	t.Parallel()

	desc := &protobufs.AgentDescription{
		IdentifyingAttributes: []*protobufs.KeyValue{
			{Key: "service.name", Value: strValue("collector")},
			{Key: "process.pid", Value: &protobufs.AnyValue{Value: &protobufs.AnyValue_IntValue{IntValue: 4321}}},
			{Key: "feature.enabled", Value: &protobufs.AnyValue{Value: &protobufs.AnyValue_BoolValue{BoolValue: true}}},
			{Key: "cpu.ratio", Value: &protobufs.AnyValue{Value: &protobufs.AnyValue_DoubleValue{DoubleValue: 0.5}}},
		},
		NonIdentifyingAttributes: nil,
	}

	got := descToDomain(desc)

	require.NotNil(t, got)
	assert.Equal(t, map[string]string{
		"service.name":    "collector",
		"process.pid":     "4321",
		"feature.enabled": "true",
		"cpu.ratio":       "0.5",
	}, got.IdentifyingAttributes)
}

func TestAnyValueToString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   *protobufs.AnyValue
		want string
	}{
		{"nil", nil, ""},
		{"string", strValue("x"), "x"},
		{"int", &protobufs.AnyValue{Value: &protobufs.AnyValue_IntValue{IntValue: -7}}, "-7"},
		{"bool", &protobufs.AnyValue{Value: &protobufs.AnyValue_BoolValue{BoolValue: false}}, "false"},
		{"double", &protobufs.AnyValue{Value: &protobufs.AnyValue_DoubleValue{DoubleValue: 1.25}}, "1.25"},
		{
			"bytes",
			&protobufs.AnyValue{Value: &protobufs.AnyValue_BytesValue{BytesValue: []byte{0x01, 0x02}}},
			"AQI=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, anyValueToString(tt.in))
		})
	}
}

// TestRemoteConfigStatusToDomain_UsesSuppliedTimestamp pins that the converter stamps the
// caller-supplied time (the injected clock) rather than reading time.Now() itself, so the
// LastUpdatedAt is deterministic.
func TestRemoteConfigStatusToDomain_UsesSuppliedTimestamp(t *testing.T) {
	t.Parallel()

	lastUpdatedAt := time.Date(2026, time.July, 21, 10, 0, 0, 0, time.UTC)

	got := remoteConfigStatusToDomain(&protobufs.RemoteConfigStatus{
		LastRemoteConfigHash: []byte{0x01},
		Status:               protobufs.RemoteConfigStatuses_RemoteConfigStatuses_APPLIED,
		ErrorMessage:         "",
	}, lastUpdatedAt)

	require.NotNil(t, got)
	assert.Equal(t, lastUpdatedAt, got.LastUpdatedAt)
}

// TestRemoteConfigStatusToDomain_NilReturnsNil keeps the nil short-circuit intact.
func TestRemoteConfigStatusToDomain_NilReturnsNil(t *testing.T) {
	t.Parallel()

	assert.Nil(t, remoteConfigStatusToDomain(nil, time.Now()))
}
