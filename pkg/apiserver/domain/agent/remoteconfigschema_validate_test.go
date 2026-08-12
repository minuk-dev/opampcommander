package agentmodel_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
)

// mapField builds a map ConfigField from named settings.
func mapField(children map[string]agentmodel.ConfigField) agentmodel.ConfigField {
	return agentmodel.ConfigField{
		Type: agentmodel.ConfigFieldTypeMap, Children: children, Open: false, Enum: nil, Doc: "",
	}
}

// scalar builds a scalar ConfigField of the given type.
func scalar(fieldType string) agentmodel.ConfigField {
	return agentmodel.ConfigField{Type: fieldType, Children: nil, Open: false, Enum: nil, Doc: ""}
}

// component builds a catalog component with the given config field schema.
func component(name string, fields agentmodel.ConfigField) agentmodel.Component {
	return agentmodel.Component{
		Type: name, Signals: nil, Stability: nil, Pairs: nil, Module: "", Fields: &fields,
	}
}

// schemaWith builds a RemoteConfigSchema whose otlp receiver has the given config field.
func schemaWith(otlp agentmodel.ConfigField) *agentmodel.RemoteConfigSchema {
	return &agentmodel.RemoteConfigSchema{
		Metadata: agentmodel.RemoteConfigSchemaMetadata{},
		Spec: agentmodel.RemoteConfigSchemaSpec{
			Binary:  "otelcol",
			Version: "0.130.0",
			Components: agentmodel.ComponentCatalog{
				"receivers": {
					"otlp": component("otlp", otlp),
					"kafka": {
						Type: "kafka", Signals: nil, Stability: nil, Pairs: nil, Module: "", Fields: nil,
					},
				},
			},
		},
		Status: agentmodel.RemoteConfigSchemaStatus{},
	}
}

func otlpSchema() *agentmodel.RemoteConfigSchema {
	return schemaWith(mapField(map[string]agentmodel.ConfigField{
		"protocols": mapField(map[string]agentmodel.ConfigField{
			"grpc": mapField(map[string]agentmodel.ConfigField{
				"endpoint":     scalar(agentmodel.ConfigFieldTypeString),
				"read_timeout": scalar(agentmodel.ConfigFieldTypeDuration),
			}),
		}),
	}))
}

func messages(errs []agentmodel.ConfigValidationError) []string {
	out := make([]string, 0, len(errs))
	for _, e := range errs {
		out = append(out, e.Error())
	}

	return out
}

func TestValidateComponent_ValidConfig(t *testing.T) {
	t.Parallel()

	config := map[string]any{
		"protocols": map[string]any{
			"grpc": map[string]any{"endpoint": "0.0.0.0:4317", "read_timeout": "5s"},
		},
	}

	errs := otlpSchema().ValidateComponent("receivers", "otlp", config)
	assert.Empty(t, errs)
}

func TestValidateComponent_UnknownField(t *testing.T) {
	t.Parallel()

	config := map[string]any{
		"protocols": map[string]any{
			"grpc": map[string]any{"endpint": "0.0.0.0:4317"}, // typo
		},
	}

	errs := otlpSchema().ValidateComponent("receivers", "otlp", config)
	require.Len(t, errs, 1)
	assert.Equal(t, "protocols.grpc.endpint: unknown field \"endpint\"", errs[0].Error())
}

func TestValidateComponent_TypeMismatch(t *testing.T) {
	t.Parallel()

	config := map[string]any{
		"protocols": map[string]any{
			"grpc": "should-be-a-map",
		},
	}

	errs := otlpSchema().ValidateComponent("receivers", "otlp", config)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "protocols.grpc: expected map")
}

func TestValidateComponent_UnknownComponent(t *testing.T) {
	t.Parallel()

	// A component the collector build does not ship is reported.
	errs := otlpSchema().ValidateComponent("receivers", "jaeger", map[string]any{})
	require.Len(t, errs, 1)
	assert.Equal(t, `unknown receiver "jaeger" for otelcol 0.130.0`, errs[0].Error())

	// A class the catalog does not describe is not.
	assert.Empty(t, otlpSchema().ValidateComponent("processors", "batch", map[string]any{}))
}

func TestValidateComponent_WithoutFieldSchemaIsExistenceOnly(t *testing.T) {
	t.Parallel()

	// kafka is in the catalog but carries no field schema: only its existence is checked.
	errs := otlpSchema().ValidateComponent("receivers", "kafka", map[string]any{"anything": 1})
	assert.Empty(t, errs)
}

func TestValidateComponent_NilAndUntypedAreValid(t *testing.T) {
	t.Parallel()

	schema := schemaWith(mapField(map[string]agentmodel.ConfigField{
		"headers": scalar(""), // untyped -> not inspected
		"tls":     mapField(map[string]agentmodel.ConfigField{"insecure": scalar(agentmodel.ConfigFieldTypeBool)}),
	}))

	config := map[string]any{
		"headers": map[string]any{"x": "y", "z": 1},
		"tls":     nil, // nil -> valid
	}

	assert.Empty(t, schema.ValidateComponent("receivers", "otlp", config))
}

// TestValidateComponent_OpenMapAcceptsAnyKey covers config the schema generator cannot
// resolve statically (a third-party config, a recursive one): it is marked open, and
// its keys must go unchecked rather than being reported as unknown.
func TestValidateComponent_OpenMapAcceptsAnyKey(t *testing.T) {
	t.Parallel()

	open := mapField(map[string]agentmodel.ConfigField{"job_name": scalar(agentmodel.ConfigFieldTypeString)})
	open.Open = true

	schema := schemaWith(mapField(map[string]agentmodel.ConfigField{"config": open}))

	config := map[string]any{"config": map[string]any{"job_name": "x", "scrape_configs": []any{1, 2}}}
	assert.Empty(t, schema.ValidateComponent("receivers", "otlp", config))
}

func TestValidateComponent_Enum(t *testing.T) {
	t.Parallel()

	compression := scalar(agentmodel.ConfigFieldTypeString)
	compression.Enum = []string{"gzip", "zstd", "none"}

	schema := schemaWith(mapField(map[string]agentmodel.ConfigField{"compression": compression}))

	assert.Empty(t, schema.ValidateComponent("receivers", "otlp", map[string]any{"compression": "zstd"}))

	errs := schema.ValidateComponent("receivers", "otlp", map[string]any{"compression": "brotli"})
	require.Len(t, errs, 1)
	assert.Equal(t, `compression: invalid value "brotli" (want one of: gzip, zstd, none)`, errs[0].Error())
}

func TestValidateComponent_List(t *testing.T) {
	t.Parallel()

	endpoints := agentmodel.ConfigField{
		Type: agentmodel.ConfigFieldTypeList,
		Children: map[string]agentmodel.ConfigField{
			agentmodel.ConfigFieldItemKey: scalar(agentmodel.ConfigFieldTypeString),
		},
		Open: false, Enum: nil, Doc: "",
	}

	schema := schemaWith(mapField(map[string]agentmodel.ConfigField{"endpoints": endpoints}))

	assert.Empty(t, schema.ValidateComponent("receivers", "otlp",
		map[string]any{"endpoints": []any{"a:1", "b:2"}}))

	errs := schema.ValidateComponent("receivers", "otlp", map[string]any{"endpoints": []any{"a:1", 2}})
	require.Len(t, errs, 1)
	assert.Equal(t, "endpoints[1]: expected string, got int", errs[0].Error())
}

func TestValidateComponent_NumberLeniency(t *testing.T) {
	t.Parallel()

	schema := schemaWith(mapField(map[string]agentmodel.ConfigField{
		"size":    scalar(agentmodel.ConfigFieldTypeInt),
		"ratio":   scalar(agentmodel.ConfigFieldTypeFloat),
		"timeout": scalar(agentmodel.ConfigFieldTypeDuration),
	}))

	// int satisfies float; string and number both satisfy duration; int stays int.
	good := map[string]any{"size": 8192, "ratio": 5, "timeout": "5s"}
	assert.Empty(t, schema.ValidateComponent("receivers", "otlp", good))

	// float with fractional part is not an int.
	bad := map[string]any{"size": 1.5}
	errs := schema.ValidateComponent("receivers", "otlp", bad)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "size: expected int")
}

func TestValidateComponent_ReportsEveryProblem(t *testing.T) {
	t.Parallel()

	schema := schemaWith(mapField(map[string]agentmodel.ConfigField{
		"endpoint": scalar(agentmodel.ConfigFieldTypeString),
		"timeout":  scalar(agentmodel.ConfigFieldTypeDuration),
	}))

	errs := schema.ValidateComponent("receivers", "otlp", map[string]any{
		"endpoint": 4317,
		"tiemout":  "5s",
	})

	msgs := messages(errs)
	assert.Len(t, msgs, 2)
	assert.Contains(t, msgs, "endpoint: expected string, got int")
	assert.Contains(t, msgs, `tiemout: unknown field "tiemout"`)
}

func signalSchema() *agentmodel.RemoteConfigSchema {
	return &agentmodel.RemoteConfigSchema{
		Metadata: agentmodel.RemoteConfigSchemaMetadata{},
		Spec: agentmodel.RemoteConfigSchemaSpec{
			Binary:  "otelcol-contrib",
			Version: "0.157.0",
			Components: agentmodel.ComponentCatalog{
				"receivers": {
					"hostmetrics": {
						Type:      "hostmetrics",
						Signals:   []string{agentmodel.SignalMetrics},
						Stability: map[string]string{agentmodel.SignalMetrics: "beta"},
						Pairs:     nil,
						Module:    "",
						Fields:    nil,
					},
					"otlp": {
						Type:      "otlp",
						Signals:   nil, // unknown signals -> every signal accepted
						Stability: nil,
						Pairs:     nil,
						Module:    "",
						Fields:    nil,
					},
				},
				"connectors": {
					"spanmetrics": {
						Type:      "spanmetrics",
						Signals:   nil,
						Stability: nil,
						Pairs: []agentmodel.SignalPair{
							{From: agentmodel.SignalTraces, To: agentmodel.SignalMetrics},
						},
						Module: "",
						Fields: nil,
					},
				},
			},
		},
		Status: agentmodel.RemoteConfigSchemaStatus{},
	}
}

func TestValidateSignalSupport(t *testing.T) {
	t.Parallel()

	schema := signalSchema()

	assert.Empty(t, schema.ValidateSignalSupport("receivers", "hostmetrics", agentmodel.SignalMetrics))

	errs := schema.ValidateSignalSupport("receivers", "hostmetrics", agentmodel.SignalLogs)
	require.Len(t, errs, 1)
	assert.Equal(t, `receiver "hostmetrics" does not support logs (supports metrics)`, errs[0].Error())

	// A component whose signals the catalog does not record supports every signal, and an
	// unknown component is left to ValidateComponent to report.
	assert.Empty(t, schema.ValidateSignalSupport("receivers", "otlp", agentmodel.SignalLogs))
	assert.Empty(t, schema.ValidateSignalSupport("receivers", "jaeger", agentmodel.SignalLogs))
}

func TestValidateConnectorPair(t *testing.T) {
	t.Parallel()

	schema := signalSchema()

	assert.Empty(t, schema.ValidateConnectorPair(
		"connectors", "spanmetrics", agentmodel.SignalTraces, agentmodel.SignalMetrics))

	errs := schema.ValidateConnectorPair(
		"connectors", "spanmetrics", agentmodel.SignalLogs, agentmodel.SignalMetrics)
	require.Len(t, errs, 1)
	assert.Equal(t, `connector "spanmetrics" does not convert logs to metrics`, errs[0].Error())
}
