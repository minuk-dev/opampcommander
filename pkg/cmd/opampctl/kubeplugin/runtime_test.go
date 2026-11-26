package kubeplugin_test

import (
	"bytes"
	"encoding/base64"
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
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/kubeplugin"
)

func TestRuntimeScope(t *testing.T) {
	t.Parallel()

	const prefix = "/api/v1/namespaces/"

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, []string{"Bearer kubernetes-api-token"}, r.Header.Values("Authorization"))
		assert.True(t, strings.HasPrefix(r.URL.Path, prefix))
		w.Header().Set("Content-Type", "application/json")

		path := strings.TrimPrefix(r.URL.Path, prefix)
		switch path {
		case "denied", "system/configmaps/denied":
			w.WriteHeader(http.StatusForbidden)
			_, err := io.WriteString(w, `{"apiVersion":"v1","kind":"Status",`+
				`"status":"Failure","reason":"Forbidden","code":403}`)
			assert.NoError(t, err)
		case "system/configmaps/anchor":
			_, err := io.WriteString(w, `{"apiVersion":"v1","kind":"ConfigMap",`+
				`"metadata":{"name":"anchor","uid":"install"}}`)
			assert.NoError(t, err)
		default:
			owner := "install"
			if path == "foreign" {
				owner = "old-install"
			}

			err := json.NewEncoder(w).Encode(corev1.Namespace{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Namespace"},
				ObjectMeta: metav1.ObjectMeta{Name: path, UID: types.UID(path + "-uid"),
					Labels: map[string]string{"opampcommander.io/installation-uid": owner}},
			})
			assert.NoError(t, err)
		}
	}))
	t.Cleanup(server.Close)

	kubeconfig := filepath.Join(t.TempDir(), "kubeconfig")
	ca := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	data := fmt.Sprintf(runtimeKubeconfig, server.URL, base64.StdEncoding.EncodeToString(ca))
	require.NoError(t, os.WriteFile(kubeconfig, []byte(data), 0o600))

	for _, tt := range []struct {
		name      string
		args      []string
		namespace string
		format    string
		wantError string
		forbidden bool
	}{
		{name: "current context", namespace: "staging"},
		{name: "other context", args: []string{"--context", "prod", "-o", "json"}, namespace: "prod", format: "json"},
		{name: "namespace override", args: []string{"-n", "collectors", "-o", "yaml"},
			namespace: "collectors", format: "yaml"},
		{name: "namespace denied", args: []string{"-n", "denied"}, wantError: "get namespace denied", forbidden: true},
		{name: "anchor denied", args: []string{"--installation-name=denied"},
			wantError: "get installation ConfigMap", forbidden: true},
		{name: "recreated anchor", args: []string{"-n", "foreign"}, wantError: "namespace is not assigned"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var output bytes.Buffer

			command := kubeplugin.NewCommand()
			command.SetOut(&output)
			command.SetErr(io.Discard)
			command.SetArgs(append([]string{"runtime-scope", "--kubeconfig", kubeconfig,
				"--installation-namespace", "system", "--installation-name", "anchor"}, tt.args...))

			err := command.ExecuteContext(t.Context())
			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
				assert.Equal(t, tt.forbidden, apierrors.IsForbidden(err))
				assert.Empty(t, output.String())

				return
			}

			require.NoError(t, err)

			switch tt.format {
			case "json":
				want := fmt.Sprintf(`{"name":%q,"namespaceUID":%q,"installationUID":"install"}`,
					tt.namespace, tt.namespace+"-uid")
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
    server: %s
    certificate-authority-data: %s
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
