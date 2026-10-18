//nolint:testpackage // Verify the create request without interactive authentication.
package agentremoteconfig

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
)

func TestRunSchemaOptions(t *testing.T) {
	t.Parallel()

	for _, skip := range []bool{false, true} {
		name := "explicit"
		if skip {
			name = "skip"
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)

				var config v1.AgentRemoteConfig
				assert.NoError(t, json.NewDecoder(r.Body).Decode(&config))
				assert.Equal(t, []string{"schema-a", "schema-b"}, config.Spec.SchemaRefs)
				assert.Equal(t, "service: {}", config.Spec.Value)
				assert.Equal(t, "keep", config.Metadata.Attributes["existing"])

				if skip {
					assert.Equal(t, "true", config.Metadata.Attributes["opampcommander.io/skip-schema-validation"])
				}

				w.Header().Set("Content-Type", "application/json")
				assert.NoError(t, json.NewEncoder(w).Encode(config))
			}))
			defer server.Close()

			opts := CommandOptions{client: client.New(server.URL), name: "cfg", namespace: "default",
				value: "service: {}", contentType: "text/yaml", formatType: "json",
				attributes: map[string]string{"existing": "keep"}, schemaRefs: []string{"schema-a", "schema-b"},
				skipSchemaValidation: skip,
			}
			cmd := &cobra.Command{}

			var output bytes.Buffer
			cmd.SetOut(&output)
			require.NoError(t, opts.Run(cmd, nil))
		})
	}
}

func TestRunRejectsEmptySchemaRef(t *testing.T) {
	t.Parallel()

	opts := CommandOptions{schemaRefs: []string{" "}}
	require.ErrorIs(t, opts.Run(&cobra.Command{}, nil), ErrEmptySchemaRef)
}
