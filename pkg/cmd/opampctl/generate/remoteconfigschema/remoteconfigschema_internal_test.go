package remoteconfigschema

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

const sampleComponents = `
buildinfo:
    command: otelcol-contrib
    description: OpenTelemetry Collector Contrib
    version: 0.110.0
receivers:
    - name: otlp
    - name: hostmetrics
processors:
    - name: batch
    - name: memory_limiter
exporters:
    - name: otlp
    - name: debug
extensions:
    - name: health_check
connectors:
    - name: forward
`

func TestParseComponents(t *testing.T) {
	t.Parallel()

	collected, err := parseComponents([]byte(sampleComponents))
	require.NoError(t, err)

	assert.Equal(t, "otelcol-contrib", collected.Command)
	assert.Equal(t, "0.110.0", collected.Version)
	assert.Equal(t, []string{"hostmetrics", "otlp"}, collected.Components["receivers"])
	assert.Equal(t, []string{"batch", "memory_limiter"}, collected.Components["processors"])
	assert.Equal(t, []string{"debug", "otlp"}, collected.Components["exporters"])
	assert.Equal(t, []string{"health_check"}, collected.Components["extensions"])
	assert.Equal(t, []string{"forward"}, collected.Components["connectors"])
}

// scalarComponents is the older `otelcol components` shape (pre-~v0.85), where each
// component is a bare scalar string rather than a `- name:` mapping.
const scalarComponents = `
buildinfo:
    command: otelcol
    version: 0.70.0
receivers:
    - otlp
    - hostmetrics
processors:
    - batch
    - memory_limiter
exporters:
    - otlp
    - logging
extensions:
    - health_check
    - zpages
`

func TestParseComponents_ScalarFormat(t *testing.T) {
	t.Parallel()

	collected, err := parseComponents([]byte(scalarComponents))
	require.NoError(t, err)

	assert.Equal(t, "otelcol", collected.Command)
	assert.Equal(t, "0.70.0", collected.Version)
	assert.Equal(t, []string{"hostmetrics", "otlp"}, collected.Components["receivers"])
	assert.Equal(t, []string{"batch", "memory_limiter"}, collected.Components["processors"])
	assert.Equal(t, []string{"logging", "otlp"}, collected.Components["exporters"])
	assert.Equal(t, []string{"health_check", "zpages"}, collected.Components["extensions"])
}

func TestBuildSchema_DefaultsFromBuildInfo(t *testing.T) {
	t.Parallel()

	collected, err := parseComponents([]byte(sampleComponents))
	require.NoError(t, err)

	opts := &CommandOptions{namespace: "default"}

	schema, err := opts.buildSchema(collected)
	require.NoError(t, err)

	assert.Equal(t, "otelcol-contrib-0.110.0", schema.Metadata.Name)
	assert.Equal(t, "otelcol-contrib", schema.Spec.Binary)
	assert.Equal(t, "0.110.0", schema.Spec.Version)
	assert.Equal(t, v1.RemoteConfigSchemaKind, schema.Kind)
}

func TestBuildSchema_OverridesAndMissingName(t *testing.T) {
	t.Parallel()

	// Overrides win over buildinfo.
	opts := &CommandOptions{
		namespace: "prod",
		binary:    "mydist",
		version:   "1.2.3",
		name:      "custom",
	}

	schema, err := opts.buildSchema(&collected{Command: "x", Version: "y", Components: v1.ComponentCatalog{}})
	require.NoError(t, err)
	assert.Equal(t, "custom", schema.Metadata.Name)
	assert.Equal(t, "prod", schema.Metadata.Namespace)
	assert.Equal(t, "mydist", schema.Spec.Binary)
	assert.Equal(t, "1.2.3", schema.Spec.Version)

	// No name derivable (no buildinfo, no --name) is an error.
	bare := &CommandOptions{namespace: "default"}
	_, err = bare.buildSchema(&collected{Command: "", Version: "", Components: v1.ComponentCatalog{}})
	require.ErrorIs(t, err, ErrNameRequired)
}

// TestGenerate_RoundTrip runs the command over stdin and checks the emitted YAML decodes
// back into a RemoteConfigSchema the same way 'create -f' loads it (yaml -> json -> struct).
func TestGenerate_RoundTrip(t *testing.T) {
	t.Parallel()

	cmd := NewCommand()
	cmd.SetArgs([]string{"--from", "-"})
	cmd.SetIn(strings.NewReader(sampleComponents))

	var out bytes.Buffer

	cmd.SetOut(&out)

	require.NoError(t, cmd.Execute())

	got := decodeAsCreateWould(t, out.Bytes())
	assert.Equal(t, v1.RemoteConfigSchemaKind, got.Kind)
	assert.Equal(t, v1.APIVersion, got.APIVersion)
	assert.Equal(t, "otelcol-contrib-0.110.0", got.Metadata.Name)
	assert.Equal(t, "otelcol-contrib", got.Spec.Binary)
	assert.Equal(t, []string{"debug", "otlp"}, got.Spec.Components["exporters"])
}

// decodeAsCreateWould mirrors how the create command loads a YAML file: YAML is decoded to
// a generic value, re-encoded as JSON, then decoded into the target so json tags are honored.
func decodeAsCreateWould(t *testing.T, data []byte) *v1.RemoteConfigSchema {
	t.Helper()

	var generic any

	require.NoError(t, yaml.Unmarshal(data, &generic))

	jsonBytes, err := json.Marshal(generic)
	require.NoError(t, err)

	var schema v1.RemoteConfigSchema

	require.NoError(t, json.Unmarshal(jsonBytes, &schema))

	return &schema
}
