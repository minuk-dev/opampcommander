//nolint:testpackage // Inject the private CLI client.
package agentgroup

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/get/internal/selectorflags"
)

func TestListByAgentStreamsPages(t *testing.T) {
	t.Parallel()

	for _, format := range []string{"json", "yaml", "short", "text"} {
		t.Run(format, func(t *testing.T) {
			t.Parallel()

			var output bytes.Buffer

			calls := 0

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++

				assert.Equal(t, "/api/v1/namespaces/team/agents/collector/agentgroups", r.URL.Path)
				assert.Equal(t, "100", r.URL.Query().Get("limit"))
				assert.Equal(t, "true", r.URL.Query().Get("includeDeleted"))
				assert.Equal(t, "env=prod", r.URL.Query().Get("labelSelector"))
				w.Header().Set("Content-Type", "application/json")

				body := `{"items":[{"metadata":{"name":"first"}}],"metadata":{"continue":"next","remainingItemCount":1}}`

				if calls == 2 {
					assert.Equal(t, "next", r.URL.Query().Get("continue"))
					assert.Contains(t, output.String(), "first", "first page must be printed before the next request")

					body = `{"items":[{"metadata":{"name":"last"}}],"metadata":{"continue":"end","remainingItemCount":0}}`
				}

				_, err := w.Write([]byte(body))
				assert.NoError(t, err)
			}))
			defer server.Close()

			options := CommandOptions{
				client: client.New(server.URL), namespace: "team", agent: "collector",
				includeDeleted: true, formatType: format,
			}
			cmd := &cobra.Command{}
			cmd.SetContext(t.Context())
			cmd.SetOut(&output)
			options.selectors.Register(cmd, selectorflags.Labels)
			require.NoError(t, cmd.Flags().Set("selector", "env=prod"))
			require.NoError(t, options.ListByAgent(cmd))
			require.Equal(t, 2, calls)
			assert.Contains(t, output.String(), "first")
			assert.Contains(t, output.String(), "last")

			if format == "json" || format == "yaml" {
				var groups []formattedAgentGroup
				if format == "json" {
					require.NoError(t, json.Unmarshal(output.Bytes(), &groups))
				} else {
					require.NoError(t, yaml.Unmarshal(output.Bytes(), &groups))
				}

				require.Len(t, groups, 2)
				assert.Equal(t, "first", groups[0].Name)
				assert.Equal(t, "last", groups[1].Name)
			}
		})
	}
}

func TestListByAgentEmpty(t *testing.T) {
	t.Parallel()

	for _, format := range []string{"json", "yaml"} {
		t.Run(format, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, err := w.Write([]byte(`{"items":[],"metadata":{"remainingItemCount":0}}`))
				assert.NoError(t, err)
			}))
			defer server.Close()

			options := CommandOptions{client: client.New(server.URL), namespace: "team", agent: "collector", formatType: format}
			cmd := &cobra.Command{}

			var output bytes.Buffer

			cmd.SetContext(t.Context())
			cmd.SetOut(&output)
			require.NoError(t, options.ListByAgent(cmd))
			require.Equal(t, "[]\n", output.String())
		})
	}
}

func TestListByAgentCursorAndRequestErrors(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name, body string
		status     int
	}{
		{name: "missing cursor", body: `{"items":[],"metadata":{"remainingItemCount":1}}`, status: http.StatusOK},
		{name: "repeated cursor",
			body: `{"items":[],"metadata":{"continue":"same","remainingItemCount":1}}`, status: http.StatusOK},
		{name: "request failure", status: http.StatusNotFound},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			calls := 0

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls++

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(test.status)
				_, err := w.Write([]byte(test.body))
				assert.NoError(t, err)
			}))
			defer server.Close()

			options := CommandOptions{client: client.New(server.URL), namespace: "team", agent: "collector", formatType: "json"}
			cmd := &cobra.Command{}

			var output bytes.Buffer
			cmd.SetOut(&output)
			cmd.SetContext(t.Context())
			err := options.ListByAgent(cmd)
			require.Error(t, err)

			if test.status == http.StatusOK {
				require.ErrorIs(t, err, ErrCommandExecutionFailed)
			} else {
				assert.Contains(t, err.Error(), "namespace \"team\"")
			}

			require.LessOrEqual(t, calls, 2, "must not loop on a broken cursor")
		})
	}
}
