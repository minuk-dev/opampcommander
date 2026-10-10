//nolint:testpackage // Exercise CLI input and dispatch without authentication setup.
package patch

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
)

func TestReadPatch(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name, input, patch, file string
		valid                    bool
	}{
		{"inline", "", `{"metadata":{"resourceVersion":"9"}}`, "", true},
		{"stdin", `{"metadata":{"labels":{"key":null}}}`, "", "-", true},
		{"null", "", "null", "", false}, {"array", "", "[]", "", false},
		{"missing", "", "", "", false}, {"invalid", "", "{", "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			data, err := readPatch(strings.NewReader(test.input), test.patch, test.file)
			if !test.valid {
				require.ErrorIs(t, err, ErrInvalidPatch)

				return
			}

			require.NoError(t, err)
			require.Equal(t, test.input+test.patch, string(data))
		})
	}
}

func TestRun(t *testing.T) {
	t.Parallel()

	for _, resource := range []string{"agentgroup", "agentpackage", "agentremoteconfig", "namespace"} {
		t.Run(resource, func(t *testing.T) {
			t.Parallel()

			patch := []byte(`{"metadata":{"resourceVersion":"1"}}`)
			calls := 0

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++

				assert.Equal(t, http.MethodPatch, r.Method)
				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				assert.Equal(t, patch, body)
				w.Header().Set("Content-Type", "application/json")
				_, err = w.Write([]byte(`{"metadata":{"resourceVersion":"2"}}`))
				assert.NoError(t, err)
			}))
			defer server.Close()

			cmd := &cobra.Command{}

			var output bytes.Buffer
			cmd.SetOut(&output)
			require.NoError(t, run(cmd, client.New(server.URL), resource, "default", "name", patch, formatter.JSON))
			require.Equal(t, 1, calls)
			require.Contains(t, output.String(), `"resourceVersion": "2"`)
		})
	}
}
