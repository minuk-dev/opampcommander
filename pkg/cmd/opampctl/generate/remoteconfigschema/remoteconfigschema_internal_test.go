package remoteconfigschema

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/samber/lo"
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
      module: go.opentelemetry.io/collector/receiver/otlpreceiver v0.110.0
      stability:
        logs: Beta
        metrics: Stable
        traces: Stable
    - name: hostmetrics
      module: github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver v0.110.0
      stability:
        metrics: Beta
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
      stability:
        traces_to_traces: Beta
        traces_to_metrics: Alpha
`

func TestParseComponents(t *testing.T) {
	t.Parallel()

	collected, err := parseComponents([]byte(sampleComponents))
	require.NoError(t, err)

	assert.Equal(t, "otelcol-contrib", collected.Command)
	assert.Equal(t, "0.110.0", collected.Version)
	assert.Equal(t, []string{"hostmetrics", "otlp"}, names(collected.Components["receivers"]))
	assert.Equal(t, []string{"batch", "memory_limiter"}, names(collected.Components["processors"]))
	assert.Equal(t, []string{"debug", "otlp"}, names(collected.Components["exporters"]))
	assert.Equal(t, []string{"health_check"}, names(collected.Components["extensions"]))
	assert.Equal(t, []string{"forward"}, names(collected.Components["connectors"]))
}

// TestParseComponents_ComponentDetail covers what a collector reports about each
// component beyond its name: its module, its stability, and — read from the stability
// keys — the signals it handles, or for a connector the conversions it supports.
func TestParseComponents_ComponentDetail(t *testing.T) {
	t.Parallel()

	collected, err := parseComponents([]byte(sampleComponents))
	require.NoError(t, err)

	otlp := collected.Components["receivers"]["otlp"]
	assert.Equal(t, "otlp", otlp.Type)
	assert.Equal(t, "go.opentelemetry.io/collector/receiver/otlpreceiver", otlp.Module)
	assert.Equal(t, []string{"logs", "metrics", "traces"}, otlp.Signals)
	assert.Equal(t, map[string]string{"logs": "beta", "metrics": "stable", "traces": "stable"}, otlp.Stability)

	assert.Equal(t, []string{"metrics"}, collected.Components["receivers"]["hostmetrics"].Signals)

	forward := collected.Components["connectors"]["forward"]
	assert.Empty(t, forward.Signals)
	assert.Equal(t, []v1.SignalPair{
		{From: "traces", To: "metrics"},
		{From: "traces", To: "traces"},
	}, forward.Pairs)
}

// names lists the component names of a class, sorted, so a catalog can be compared
// without spelling out every component.
func names(components map[string]v1.Component) []string {
	out := lo.Keys(components)
	slices.Sort(out)

	return out
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
	assert.Equal(t, []string{"hostmetrics", "otlp"}, names(collected.Components["receivers"]))
	assert.Equal(t, []string{"batch", "memory_limiter"}, names(collected.Components["processors"]))
	assert.Equal(t, []string{"logging", "otlp"}, names(collected.Components["exporters"]))
	assert.Equal(t, []string{"health_check", "zpages"}, names(collected.Components["extensions"]))
}

func TestBuildSchema_DefaultsFromBuildInfo(t *testing.T) {
	t.Parallel()

	collected, err := parseComponents([]byte(sampleComponents))
	require.NoError(t, err)

	opts := &CommandOptions{namespace: "default", version: VersionLatest}

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
	bare := &CommandOptions{namespace: "default", version: VersionLatest}
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
	assert.Equal(t, []string{"debug", "otlp"}, names(got.Spec.Components["exporters"]))
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

// registryFixture writes a two-file registry (an index and one schema) to a temp
// directory, so the registry path can be exercised without reaching the network.
func registryFixture(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.json"),
		[]byte(`{"distributions":{"contrib":["v0.110.0","v0.157.0"],"core":["v0.157.0"]}}`), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "contrib"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "contrib", "v0.157.0.json"), []byte(`{
	  "collectorVersion": "v0.157.0",
	  "distribution": "contrib",
	  "components": {
	    "receiver": {
	      "otlp": {
	        "type": "otlp",
	        "signals": ["traces", "metrics"],
	        "stability": {"traces": "stable"},
	        "module": "go.opentelemetry.io/collector/receiver/otlpreceiver",
	        "fields": {
	          "type": "map",
	          "doc": "Config defines configuration for the OTLP receiver.",
	          "children": {
	            "compression": {"type": "string", "enum": ["gzip", "none"], "doc": "wire compression"},
	            "headers": {"type": "map", "open": true}
	          }
	        }
	      }
	    },
	    "connector": {
	      "forward": {
	        "type": "forward",
	        "pairs": [{"from": "traces", "to": "traces"}],
	        "module": "go.opentelemetry.io/collector/connector/forwardconnector"
	      }
	    }
	  }
	}`), 0o600))

	return dir
}

// TestGenerate_FromRegistry covers reading a published release from the registry: the
// newest version is selected from the index, the registry's singular class keys become
// the collector's config sections, and the settings each component accepts come along.
func TestGenerate_FromRegistry(t *testing.T) {
	t.Parallel()

	cmd := NewCommand()
	cmd.SetArgs([]string{"--distribution", "contrib", "--schema-location", registryFixture(t)})

	var out bytes.Buffer

	cmd.SetOut(&out)

	require.NoError(t, cmd.Execute())

	got := decodeAsCreateWould(t, out.Bytes())
	assert.Equal(t, "otelcol-contrib-0.157.0", got.Metadata.Name)
	assert.Equal(t, "otelcol-contrib", got.Spec.Binary)
	assert.Equal(t, "0.157.0", got.Spec.Version)

	otlp := got.Spec.Components["receivers"]["otlp"]
	assert.Equal(t, []string{"traces", "metrics"}, otlp.Signals)
	require.NotNil(t, otlp.Fields)
	assert.Equal(t, []string{"gzip", "none"}, otlp.Fields.Children["compression"].Enum)
	assert.True(t, otlp.Fields.Children["headers"].Open)
	assert.Equal(t, "wire compression", otlp.Fields.Children["compression"].Doc)

	assert.Equal(t, []v1.SignalPair{{From: "traces", To: "traces"}},
		got.Spec.Components["connectors"]["forward"].Pairs)
}

// TestGenerate_FromRegistry_StripDocs covers the form the pre-built schema library
// ships in: everything validation needs, none of the documentation.
func TestGenerate_FromRegistry_StripDocs(t *testing.T) {
	t.Parallel()

	cmd := NewCommand()
	cmd.SetArgs([]string{
		"--distribution", "contrib", "--version", "v0.157.0",
		"--schema-location", registryFixture(t), "--strip-docs",
	})

	var out bytes.Buffer

	cmd.SetOut(&out)

	require.NoError(t, cmd.Execute())
	assert.NotContains(t, out.String(), "wire compression")

	got := decodeAsCreateWould(t, out.Bytes())
	otlp := got.Spec.Components["receivers"]["otlp"]
	require.NotNil(t, otlp.Fields)
	assert.Empty(t, otlp.Fields.Doc)
	assert.Equal(t, []string{"gzip", "none"}, otlp.Fields.Children["compression"].Enum)
}

func TestGenerate_FromRegistry_UnknownDistribution(t *testing.T) {
	t.Parallel()

	cmd := NewCommand()
	cmd.SetArgs([]string{"--distribution", "nope", "--schema-location", registryFixture(t)})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	require.ErrorIs(t, cmd.Execute(), ErrNoSchemaForDistribution)
}
