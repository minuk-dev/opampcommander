package remoteconfigschema_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/create/remoteconfigschema"
	generate "github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/generate/remoteconfigschema"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

const componentsOutput = `
buildinfo:
  command: otelcol-contrib
  version: 0.110.0
receivers:
  - name: otlp
    module: go.opentelemetry.io/collector/receiver/otlpreceiver v0.110.0
    stability:
      traces: Stable
processors:
  - name: batch
exporters:
  - name: debug
extensions:
  - name: health_check
connectors:
  - name: forward
    stability:
      traces_to_metrics: Alpha
`

func TestGenerateCreateRoundTrip(t *testing.T) {
	t.Parallel()

	for _, format := range []string{"yaml", "json"} {
		t.Run(format, func(t *testing.T) {
			t.Parallel()

			generateCmd := generate.NewCommand()
			generateCmd.SetArgs([]string{"--from", "-", "--namespace", "collectors", "-o", format})
			generateCmd.SetIn(strings.NewReader(componentsOutput))

			var manifest bytes.Buffer
			generateCmd.SetOut(&manifest)
			require.NoError(t, generateCmd.ExecuteContext(t.Context()))

			file := filepath.Join(t.TempDir(), "schema."+format)
			require.NoError(t, os.WriteFile(file, manifest.Bytes(), 0o600))

			requests := make(chan v1.RemoteConfigSchema, 1)

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/api/v1/namespaces/collectors/remoteconfigschemas", r.URL.Path)

				var schema v1.RemoteConfigSchema

				err := json.NewDecoder(r.Body).Decode(&schema)
				if !assert.NoError(t, err) {
					http.Error(w, "invalid schema", http.StatusBadRequest)

					return
				}

				requests <- schema

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				assert.NoError(t, json.NewEncoder(w).Encode(schema))
			}))
			defer server.Close()

			conf := config.NewDefaultGlobalConfig(t.TempDir())
			conf.Clusters[0].OpAMPCommander.Endpoint = server.URL
			conf.Users[0].Auth.Type = config.AuthTypeManual
			conf.Users[0].Auth.BearerToken = "test-token"
			conf.Log.Logger = slog.New(slog.DiscardHandler)

			createCmd := remoteconfigschema.NewCommand(remoteconfigschema.CommandOptions{GlobalConfig: conf})
			createCmd.SetArgs([]string{"-f", file, "-o", "json"})
			createCmd.SetOut(io.Discard)
			require.NoError(t, createCmd.ExecuteContext(t.Context()))
			require.Len(t, requests, 1, "create must submit the generated manifest")

			got := <-requests
			assert.Equal(t, v1.RemoteConfigSchemaKind, got.Kind)
			assert.Equal(t, v1.APIVersion, got.APIVersion)
			assert.Equal(t, "otelcol-contrib-0.110.0", got.Metadata.Name)
			assert.Equal(t, "collectors", got.Metadata.Namespace)
			assert.Equal(t, "otelcol-contrib", got.Spec.Binary)
			assert.Equal(t, "0.110.0", got.Spec.Version)
			assert.Equal(t, v1.ComponentCatalog{
				"receivers": {"otlp": {
					Type: "otlp", Module: "go.opentelemetry.io/collector/receiver/otlpreceiver",
					Signals: []string{"traces"}, Stability: map[string]string{"traces": "stable"},
				}},
				"processors": {"batch": {Type: "batch"}},
				"exporters":  {"debug": {Type: "debug"}},
				"extensions": {"health_check": {Type: "health_check"}},
				"connectors": {"forward": {
					Type: "forward", Stability: map[string]string{"traces_to_metrics": "alpha"},
					Pairs: []v1.SignalPair{{From: "traces", To: "metrics"}},
				}},
			}, got.Spec.Components)
		})
	}
}
