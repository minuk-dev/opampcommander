package client_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/client"
)

func TestPatchRequests(t *testing.T) {
	t.Parallel()

	for _, resource := range []string{"agentgroup", "agentpackage", "agentremoteconfig", "namespace"} {
		t.Run(resource, func(t *testing.T) {
			t.Parallel()

			for _, patch := range []string{`{"metadata":{"attributes":{"key":null}}}`,
				`{"metadata":{"resourceVersion":"123"}}`} {
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++

					assert.Equal(t, http.MethodPatch, r.Method)

					path := "/api/v1/namespaces/ns/" + resource + "s/name"
					if resource == "namespace" {
						path = "/api/v1/namespaces/name"
					}

					assert.Equal(t, path, r.URL.Path)
					assert.Equal(t, "application/merge-patch+json", r.Header.Get("Content-Type"))
					data, err := io.ReadAll(r.Body)
					assert.NoError(t, err)
					assert.Equal(t, patch, string(data))
					w.Header().Set("Content-Type", "application/json")
					_, err = w.Write([]byte(`{"metadata":{"resourceVersion":"124"}}`))
					assert.NoError(t, err)
				}))
				cli := client.New(server.URL)

				var version int64

				switch resource {
				case "agentgroup":
					v, err := cli.AgentGroupService.PatchAgentGroup(t.Context(), "ns", "name", []byte(patch))
					require.NoError(t, err)

					version = v.Metadata.ResourceVersion
				case "agentpackage":
					v, err := cli.AgentPackageService.PatchAgentPackage(t.Context(), "ns", "name", []byte(patch))
					require.NoError(t, err)

					version = v.Metadata.ResourceVersion
				case "agentremoteconfig":
					v, err := cli.AgentRemoteConfigService.PatchAgentRemoteConfig(t.Context(), "ns", "name",
						[]byte(patch))
					require.NoError(t, err)

					version = v.Metadata.ResourceVersion
				case "namespace":
					v, err := cli.NamespaceService.PatchNamespace(t.Context(), "name", []byte(patch))
					require.NoError(t, err)

					version = v.Metadata.ResourceVersion
				}

				server.Close()
				require.EqualValues(t, 124, version)
				require.Equal(t, 1, calls)
			}
		})
	}
}
