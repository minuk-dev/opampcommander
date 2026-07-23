//nolint:testpackage // white-box test of the unexported protobuf->domain converters
package opamp

import (
	"testing"

	"github.com/open-telemetry/opamp-go/protobufs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/timeutil"
)

func TestDescToDomain_Nil(t *testing.T) {
	t.Parallel()
	assert.Nil(t, descToDomain(nil))
}

// TestAnyValueToString_NestedAndUnknown covers the fallback branches: array/kvlist values fall
// back to their protobuf text form (non-empty), and an AnyValue with no oneof set yields "".
func TestAnyValueToString_NestedAndUnknown(t *testing.T) {
	t.Parallel()

	arr := &protobufs.AnyValue{Value: &protobufs.AnyValue_ArrayValue{
		ArrayValue: &protobufs.ArrayValue{Values: []*protobufs.AnyValue{strValue("a")}},
	}}
	assert.NotEmpty(t, anyValueToString(arr), "array value should fall back to protobuf text form")

	kv := &protobufs.AnyValue{Value: &protobufs.AnyValue_KvlistValue{
		KvlistValue: &protobufs.KeyValueList{Values: []*protobufs.KeyValue{{Key: "k", Value: strValue("v")}}},
	}}
	assert.NotEmpty(t, anyValueToString(kv), "kvlist value should fall back to protobuf text form")

	assert.Empty(t, anyValueToString(&protobufs.AnyValue{}), "unset value should render as empty string")
}

func TestConnectionSettingsStatusToDomain(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, connectionSettingsStatusToDomain(nil))
	})

	t.Run("maps every field", func(t *testing.T) {
		t.Parallel()

		got := connectionSettingsStatusToDomain(&protobufs.ConnectionSettingsStatus{
			LastConnectionSettingsHash: []byte{0xAB, 0xCD},
			Status:                     protobufs.ConnectionSettingsStatuses_ConnectionSettingsStatuses_FAILED,
			ErrorMessage:               "boom",
		})

		require.NotNil(t, got)
		assert.Equal(t, []byte{0xAB, 0xCD}, got.LastConnectionSettingsHash)
		assert.Equal(t, agentmodel.ConnectionSettingsStatusFailed, got.Status)
		assert.Equal(t, "boom", got.ErrorMessage)
	})
}

func TestCustomCapabilitiesToDomain(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, customCapabilitiesToDomain(nil))
	})

	t.Run("copies capability list", func(t *testing.T) {
		t.Parallel()

		got := customCapabilitiesToDomain(&protobufs.CustomCapabilities{
			Capabilities: []string{"io.opentelemetry.foo", "io.opentelemetry.bar"},
		})

		require.NotNil(t, got)
		assert.Equal(t, []string{"io.opentelemetry.foo", "io.opentelemetry.bar"}, got.Capabilities)
	})
}

func TestHealthToDomain(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, healthToDomain(nil))
	})

	t.Run("maps fields and nests sub-component health", func(t *testing.T) {
		t.Parallel()

		const (
			startNanos  = uint64(1_000_000_000)
			statusNanos = uint64(2_500_000_000)
			subNanos    = uint64(3_000_000_000)
		)

		got := healthToDomain(&protobufs.ComponentHealth{
			Healthy:            true,
			StartTimeUnixNano:  startNanos,
			LastError:          "",
			Status:             "running",
			StatusTimeUnixNano: statusNanos,
			ComponentHealthMap: map[string]*protobufs.ComponentHealth{
				"receiver/otlp": {
					Healthy:            false,
					StartTimeUnixNano:  subNanos,
					LastError:          "listen failed",
					Status:             "error",
					StatusTimeUnixNano: subNanos,
				},
			},
		})

		require.NotNil(t, got)
		assert.True(t, got.Healthy)
		assert.Equal(t, timeutil.UnixNanoToTime(startNanos), got.StartTime)
		assert.Equal(t, "running", got.Status)
		assert.Equal(t, timeutil.UnixNanoToTime(statusNanos), got.StatusTime)

		require.Contains(t, got.ComponentHealthMap, "receiver/otlp")
		sub := got.ComponentHealthMap["receiver/otlp"]
		assert.False(t, sub.Healthy)
		assert.Equal(t, "listen failed", sub.LastError)
		assert.Equal(t, "error", sub.Status)
	})
}

func TestEffectiveConfigToDomain(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, effectiveConfigToDomain(nil))
	})

	t.Run("maps config files", func(t *testing.T) {
		t.Parallel()

		got := effectiveConfigToDomain(&protobufs.EffectiveConfig{
			ConfigMap: &protobufs.AgentConfigMap{
				ConfigMap: map[string]*protobufs.AgentConfigFile{
					"otel.yaml": {
						Body:        []byte("receivers: {}"),
						ContentType: "application/yaml",
					},
				},
			},
		})

		require.NotNil(t, got)
		require.Contains(t, got.ConfigMap.ConfigMap, "otel.yaml")
		file := got.ConfigMap.ConfigMap["otel.yaml"]
		assert.Equal(t, []byte("receivers: {}"), file.Body)
		assert.Equal(t, "application/yaml", file.ContentType)
	})
}

func TestPackageStatusToDomain(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, packageStatusToDomain(nil))
	})

	t.Run("maps package entries and top-level fields", func(t *testing.T) {
		t.Parallel()

		got := packageStatusToDomain(&protobufs.PackageStatuses{
			Packages: map[string]*protobufs.PackageStatus{
				"collector": {
					Name:                 "collector",
					AgentHasVersion:      "0.115.1",
					AgentHasHash:         []byte{0x01},
					ServerOfferedVersion: "0.116.0",
					Status:               protobufs.PackageStatusEnum_PackageStatusEnum_Installing,
					ErrorMessage:         "",
				},
			},
			ServerProvidedAllPackagesHash: []byte{0x0A, 0x0B},
			ErrorMessage:                  "partial",
		})

		require.NotNil(t, got)
		assert.Equal(t, []byte{0x0A, 0x0B}, got.ServerProvidedAllPackgesHash)
		assert.Equal(t, "partial", got.ErrorMessage)

		require.Contains(t, got.Packages, "collector")
		entry := got.Packages["collector"]
		assert.Equal(t, "collector", entry.Name)
		assert.Equal(t, "0.115.1", entry.AgentHasVersion)
		assert.Equal(t, "0.116.0", entry.ServerOfferedVersion)
		assert.Equal(t, agentmodel.AgentPackageStatusEnum(agentmodel.AgentPackageStatusEnumInstalling), entry.Status)
	})
}

func TestAvailableComponentsToDomain(t *testing.T) {
	t.Parallel()

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, availableComponentsToDomain(nil))
	})

	t.Run("maps components, metadata and nested sub-components", func(t *testing.T) {
		t.Parallel()

		got := availableComponentsToDomain(&protobufs.AvailableComponents{
			Hash: []byte{0xFF},
			Components: map[string]*protobufs.ComponentDetails{
				"receivers": {
					Metadata: []*protobufs.KeyValue{
						{Key: "code.namespace", Value: strValue("otlpreceiver")},
					},
					SubComponentMap: map[string]*protobufs.ComponentDetails{
						"otlp": {
							Metadata: []*protobufs.KeyValue{
								{Key: "stability", Value: strValue("stable")},
							},
						},
					},
				},
			},
		})

		require.NotNil(t, got)
		assert.Equal(t, []byte{0xFF}, got.Hash)

		require.Contains(t, got.Components, "receivers")
		receivers := got.Components["receivers"]
		assert.Equal(t, "otlpreceiver", receivers.Metadata["code.namespace"])

		require.Contains(t, receivers.SubComponentMap, "otlp")
		assert.Equal(t, "stable", receivers.SubComponentMap["otlp"].Metadata["stability"])
	})
}
