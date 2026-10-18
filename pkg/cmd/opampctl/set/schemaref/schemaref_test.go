//nolint:testpackage // Inspect CLI payloads without interactive authentication.
package schemaref

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/client"
)

func TestRun(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name   string
		clear  bool
		status int
	}{
		{"replace", false, http.StatusOK}, {"clear", true, http.StatusOK},
		{"conflict", false, http.StatusConflict},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v1/namespaces/ns/agentremoteconfigs/cfg", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")

				if r.Method == http.MethodGet {
					_, _ = w.Write([]byte(`{"metadata":{"resourceVersion":"7"},"spec":{"value":"keep this", "schemaRefs":["old"]}}`))

					return
				}

				assert.Equal(t, http.MethodPatch, r.Method)
				assert.Equal(t, "application/merge-patch+json", r.Header.Get("Content-Type"))

				var patch map[string]any
				assert.NoError(t, json.NewDecoder(r.Body).Decode(&patch))

				refs := []any{"new", "second"}
				if tt.clear {
					refs = []any{}
				}

				assert.Equal(t, map[string]any{
					"metadata": map[string]any{"resourceVersion": "7"},
					"spec":     map[string]any{"schemaRefs": refs},
				}, patch)
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(`{"metadata":{"name":"cfg","resourceVersion":"8"}}`))
			}))
			defer server.Close()

			opts := CommandOptions{client: client.New(server.URL), namespace: "ns", clear: tt.clear, formatType: "json"}
			if !tt.clear {
				opts.schemaRefs = []string{"new", "second"}
			}

			cmd := &cobra.Command{}

			var output bytes.Buffer
			cmd.SetOut(&output)

			err := opts.Run(cmd, []string{"cfg"})
			if tt.status != http.StatusOK {
				require.Error(t, err)
				assert.Empty(t, output.String())
			} else {
				require.NoError(t, err)
				assert.Contains(t, output.String(), `"resourceVersion": "8"`)
			}
		})
	}
}

func TestCommandRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name string
		args []string
	}{
		{"missing action", []string{"cfg"}},
		{"empty reference", []string{"cfg", "--schema-ref="}},
		{"conflicting actions", []string{"cfg", "--clear", "--schema-ref=a"}},
		{"false clear", []string{"cfg", "--clear=false"}},
		{"missing name", []string{"--clear"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := NewCommand(CommandOptions{})
			cmd.SetArgs(tt.args)
			require.Error(t, cmd.Execute())
		})
	}
}
