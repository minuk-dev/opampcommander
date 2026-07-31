package agentmodel_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
)

// obj builds an object ConfigField from named sub-fields.
func obj(fields map[string]agentmodel.ConfigField) agentmodel.ConfigField {
	return agentmodel.ConfigField{Type: agentmodel.ConfigFieldTypeObject, Fields: fields, Elem: nil}
}

// scalar builds a scalar ConfigField of the given type.
func scalar(t string) agentmodel.ConfigField {
	return agentmodel.ConfigField{Type: t, Fields: nil, Elem: nil}
}

// schemaWith builds a RemoteConfigSchema whose otlp receiver has the given config field.
func schemaWith(otlp agentmodel.ConfigField) *agentmodel.RemoteConfigSchema {
	return &agentmodel.RemoteConfigSchema{
		Metadata: agentmodel.RemoteConfigSchemaMetadata{},
		Spec: agentmodel.RemoteConfigSchemaSpec{
			Binary:     "otelcol",
			Version:    "0.130.0",
			Components: agentmodel.ComponentCatalog{"receivers": {"otlp"}},
			ComponentConfigs: agentmodel.ComponentConfigCatalog{
				"receivers": {"otlp": otlp},
			},
		},
		Status: agentmodel.RemoteConfigSchemaStatus{},
	}
}

func otlpSchema() *agentmodel.RemoteConfigSchema {
	return schemaWith(obj(map[string]agentmodel.ConfigField{
		"protocols": obj(map[string]agentmodel.ConfigField{
			"grpc": obj(map[string]agentmodel.ConfigField{
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

func TestValidateComponentConfig_ValidConfig(t *testing.T) {
	t.Parallel()

	config := map[string]any{
		"protocols": map[string]any{
			"grpc": map[string]any{"endpoint": "0.0.0.0:4317", "read_timeout": "5s"},
		},
	}

	errs := otlpSchema().ValidateComponentConfig("receivers", "otlp", config)
	assert.Empty(t, errs)
}

func TestValidateComponentConfig_UnknownField(t *testing.T) {
	t.Parallel()

	config := map[string]any{
		"protocols": map[string]any{
			"grpc": map[string]any{"endpint": "0.0.0.0:4317"}, // typo
		},
	}

	errs := otlpSchema().ValidateComponentConfig("receivers", "otlp", config)
	require.Len(t, errs, 1)
	assert.Equal(t, "protocols.grpc.endpint: unknown field \"endpint\"", errs[0].Error())
}

func TestValidateComponentConfig_TypeMismatch(t *testing.T) {
	t.Parallel()

	config := map[string]any{
		"protocols": map[string]any{
			"grpc": "should-be-an-object",
		},
	}

	errs := otlpSchema().ValidateComponentConfig("receivers", "otlp", config)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "protocols.grpc: expected object")
}

func TestValidateComponentConfig_UnknownComponentIsExistenceOnly(t *testing.T) {
	t.Parallel()

	// No config schema for this component -> nil (only existence would be checked elsewhere).
	errs := otlpSchema().ValidateComponentConfig("receivers", "kafka", map[string]any{"anything": 1})
	assert.Nil(t, errs)

	// Unknown class -> nil too.
	errs = otlpSchema().ValidateComponentConfig("processors", "otlp", map[string]any{})
	assert.Nil(t, errs)
}

func TestValidateComponentConfig_NilAndAnyAreValid(t *testing.T) {
	t.Parallel()

	schema := schemaWith(obj(map[string]agentmodel.ConfigField{
		"headers": scalar(agentmodel.ConfigFieldTypeAny),
		"tls":     obj(map[string]agentmodel.ConfigField{"insecure": scalar(agentmodel.ConfigFieldTypeBool)}),
	}))

	config := map[string]any{
		"headers": map[string]any{"x": "y", "z": 1}, // any -> not inspected
		"tls":     nil,                              // nil -> valid
	}

	assert.Empty(t, schema.ValidateComponentConfig("receivers", "otlp", config))
}

func TestValidateComponentConfig_MapAndArray(t *testing.T) {
	t.Parallel()

	strScalar := scalar(agentmodel.ConfigFieldTypeString)
	schema := schemaWith(obj(map[string]agentmodel.ConfigField{
		// headers: map[string]string -> arbitrary keys, string values
		"headers": {Type: agentmodel.ConfigFieldTypeMap, Fields: nil, Elem: &strScalar},
		// endpoints: []string
		"endpoints": {Type: agentmodel.ConfigFieldTypeArray, Fields: nil, Elem: &strScalar},
	}))

	ok := map[string]any{
		"headers":   map[string]any{"authorization": "bearer", "x-custom": "v"},
		"endpoints": []any{"a:1", "b:2"},
	}
	assert.Empty(t, schema.ValidateComponentConfig("receivers", "otlp", ok))

	bad := map[string]any{
		"headers":   map[string]any{"authorization": 123}, // value not string
		"endpoints": []any{"a:1", 2},                      // element not string
	}
	errs := schema.ValidateComponentConfig("receivers", "otlp", bad)
	msgs := messages(errs)
	assert.Len(t, msgs, 2)
	assert.Contains(t, msgs, "headers.authorization: expected string, got int")
	assert.Contains(t, msgs, "endpoints[1]: expected string, got int")
}

func TestValidateComponentConfig_NumberLeniency(t *testing.T) {
	t.Parallel()

	schema := schemaWith(obj(map[string]agentmodel.ConfigField{
		"size":    scalar(agentmodel.ConfigFieldTypeInt),
		"ratio":   scalar(agentmodel.ConfigFieldTypeFloat),
		"timeout": scalar(agentmodel.ConfigFieldTypeDuration),
	}))

	// int satisfies float; string and number both satisfy duration; int stays int.
	good := map[string]any{"size": 8192, "ratio": 5, "timeout": "5s"}
	assert.Empty(t, schema.ValidateComponentConfig("receivers", "otlp", good))

	// float with fractional part is not an int.
	bad := map[string]any{"size": 1.5}
	errs := schema.ValidateComponentConfig("receivers", "otlp", bad)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Error(), "size: expected int")
}
