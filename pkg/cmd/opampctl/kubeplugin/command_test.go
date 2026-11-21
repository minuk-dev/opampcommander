package kubeplugin_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/kubeplugin"
)

func TestValidateRedactsAdmissionDiagnostics(t *testing.T) {
	t.Parallel()

	const private = "private-config-marker"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/apis/opampcommander.io/v1alpha1/namespaces/team/remoteconfigs/example", r.URL.Path)
		assert.Equal(t, "All", r.URL.Query().Get("dryRun"))
		assert.Equal(t, "Strict", r.URL.Query().Get("fieldValidation"))
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Warning", `299 admission "private-config-marker"`)
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = fmt.Fprint(w, `{"apiVersion":"v1","kind":"Status","status":"Failure","code":422,
"reason":"Invalid","message":"private-config-marker"}`)
	}))
	defer server.Close()

	config := filepath.Join(t.TempDir(), "config")
	require.NoError(t, os.WriteFile(config, []byte(fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- name: c
  cluster: {server: %s}
contexts:
- name: team
  context: {cluster: c, namespace: team}
current-context: team
`, server.URL)), 0o600))

	cmd := kubeplugin.NewCommand()
	cmd.SetArgs([]string{"validate", "--kubeconfig", config, "-f", "-"})
	cmd.SetIn(strings.NewReader(`apiVersion: opampcommander.io/v1alpha1
kind: RemoteConfig
metadata: {name: example}
spec: {value: private-config-marker}
`))

	var output bytes.Buffer
	cmd.SetErr(&output)
	cmd.SetOut(&output)
	err := cmd.ExecuteContext(t.Context())
	require.ErrorContains(t, err, "HTTP 422")
	require.NotContains(t, err.Error(), private)
	require.NotContains(t, output.String(), private)
}
