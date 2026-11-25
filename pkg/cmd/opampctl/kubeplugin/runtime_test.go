package kubeplugin_test

import (
	"bytes"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/kubeplugin"
)

func TestRuntimeScope(t *testing.T) {
	t.Parallel()

	const prefix = "/api/runtime/v1/namespaces/"

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, []string{"Bearer runtime-audience-token"}, r.Header.Values("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.True(t, strings.HasPrefix(r.URL.Path, prefix))
		namespace := strings.TrimPrefix(r.URL.Path, prefix)

		switch namespace {
		case "denied":
			http.Error(w, "private failure detail", http.StatusForbidden)
		case "redirected":
			w.Header().Set("Location", prefix+"staging")
			w.WriteHeader(http.StatusFound)
		case "invalid-response":
			_, err := io.WriteString(w, `{"name":"other","namespaceUID":"foreign","installationUID":"install"}`)
			assert.NoError(t, err)
		default:
			assert.NoError(t, json.NewEncoder(w).Encode(map[string]string{
				"name": namespace, "namespaceUID": namespace + "-uid", "installationUID": "install",
			}))
		}
	}))
	t.Cleanup(server.Close)

	dir := t.TempDir()
	kubeconfig := filepath.Join(dir, "kubeconfig")
	tokenFile := filepath.Join(dir, "runtime-token")
	caFile := filepath.Join(dir, "runtime-ca.pem")

	require.NoError(t, os.WriteFile(kubeconfig, []byte(runtimeKubeconfig), 0o600))
	require.NoError(t, os.WriteFile(tokenFile, []byte("runtime-audience-token\n"), 0o600))
	require.NoError(t, os.WriteFile(caFile, pem.EncodeToMemory(&pem.Block{
		Type: "CERTIFICATE", Bytes: server.Certificate().Raw,
	}), 0o600))

	for _, tt := range []struct {
		name      string
		args      []string
		namespace string
		format    string
		wantError string
	}{
		{name: "current context", namespace: "staging"},
		{name: "other context", args: []string{"--context", "prod", "-o", "json"}, namespace: "prod", format: "json"},
		{name: "namespace override", args: []string{"-n", "collectors", "-o", "yaml"},
			namespace: "collectors", format: "yaml"},
		{name: "denied", args: []string{"-n", "denied"}, wantError: "HTTP 403 Forbidden"},
		{name: "redirect", args: []string{"-n", "redirected"}, wantError: "HTTP 302 Found"},
		{name: "invalid response", args: []string{"-n", "invalid-response"}, wantError: "invalid namespace scope"},
		{name: "untrusted certificate", args: []string{"--runtime-ca-file="},
			wantError: "certificate signed by unknown authority"},
		{name: "plaintext origin", args: []string{"--runtime-url=http://" + strings.TrimPrefix(server.URL, "https://")},
			wantError: "must be an HTTPS origin"},
		{name: "origin with query", args: []string{"--runtime-url=" + server.URL + "?value=private"},
			wantError: "must be an HTTPS origin"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var output bytes.Buffer

			command := kubeplugin.NewCommand()
			command.SetOut(&output)
			command.SetErr(io.Discard)
			command.SetArgs(append([]string{"runtime-scope", "--kubeconfig", kubeconfig,
				"--runtime-url", server.URL, "--runtime-ca-file", caFile, "--runtime-token-file", tokenFile}, tt.args...))

			err := command.ExecuteContext(t.Context())
			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
				assert.NotContains(t, err.Error(), "private")
				assert.Empty(t, output.String())

				return
			}

			require.NoError(t, err)

			want := fmt.Sprintf(`{"name":%q,"namespaceUID":%q,"installationUID":"install"}`, tt.namespace, tt.namespace+"-uid")
			switch tt.format {
			case "json":
				assert.JSONEq(t, want, output.String())
			case "yaml":
				var scope map[string]string

				require.NoError(t, yaml.Unmarshal(output.Bytes(), &scope))
				assert.Equal(t, map[string]string{
					"name": tt.namespace, "namespaceUID": tt.namespace + "-uid", "installationUID": "install",
				}, scope)
			default:
				assert.Contains(t, output.String(), "INSTALLATION UID")
				assert.Contains(t, output.String(), tt.namespace+"-uid")
			}
		})
	}
}

const runtimeKubeconfig = `apiVersion: v1
kind: Config
clusters:
- name: kube-api
  cluster:
    server: https://kubernetes.example.invalid
users:
- name: human
  user:
    token: kubernetes-api-token
contexts:
- name: staging
  context:
    cluster: kube-api
    user: human
    namespace: staging
- name: prod
  context:
    cluster: kube-api
    user: human
    namespace: prod
current-context: staging
`
