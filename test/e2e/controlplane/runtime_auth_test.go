//go:build kind_e2e

package controlplane_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func testRuntimeAuthorization(t *testing.T, port int) {
	t.Helper()

	installationUID := string(kubectl(t, "get", "configmap", "-n", operatorNamespace,
		"opampcommander-operator-opamp-installation", "-o", "jsonpath={.metadata.uid}"))
	kubectl(t, "create", "serviceaccount", "runtime-reader", "-n", "opamp-e2e")
	kubectl(t, "create", "namespace", "runtime-tenant")
	assignRuntimeNamespace(t, installationUID)
	grantRuntimeReader(t)
	token := runtimeToken(t, "opampcommander-runtime")

	require.Eventually(t, func() bool {
		status, _ := runtimeRequest(t, port, "runtime-tenant", token)

		return status == http.StatusOK
	}, timeout, poll, "scoped ServiceAccount must resolve its assigned namespace")
	_, scope := runtimeRequest(t, port, "runtime-tenant", token)
	require.Equal(t, installationUID, scope["installationUID"])
	require.Equal(t, string(kubectl(t, "get", "namespace", "runtime-tenant", "-o", "jsonpath={.metadata.uid}")),
		scope["namespaceUID"])

	status, _ := runtimeRequest(t, port, "runtime-tenant", "")
	require.Equal(t, http.StatusUnauthorized, status, "forwarded admin headers cannot authenticate")
	status, _ = runtimeRequest(t, port, "runtime-tenant", runtimeToken(t, "wrong-audience"))
	require.Equal(t, http.StatusUnauthorized, status)
	status, _ = runtimeRequest(t, port, "default", token)
	require.Equal(t, http.StatusForbidden, status, "a RoleBinding grants access in only its namespace")
	status, _ = runtimeRequest(t, port, "runtime-tenant", "legacy-or-invalid-jwt")
	require.Equal(t, http.StatusUnauthorized, status)

	kubectl(t, "label", "namespace", "runtime-tenant", "opampcommander.io/installation-uid=foreign", "--overwrite")
	status, _ = runtimeRequest(t, port, "runtime-tenant", token)
	require.Equal(t, http.StatusForbidden, status, "permission does not override installation ownership")
	assignRuntimeNamespace(t, installationUID)

	kubectl(t, "delete", "rolebinding", "runtime-reader", "-n", "runtime-tenant")
	require.Eventually(t, func() bool {
		status, _ := runtimeRequest(t, port, "runtime-tenant", token)

		return status == http.StatusForbidden
	}, timeout, poll, "permission revocation must affect an already used token")

	kubectl(t, "delete", "namespace", "runtime-tenant", "--wait=true", "--timeout=90s")
	kubectl(t, "create", "namespace", "runtime-tenant")
	grantRuntimeReader(t)
	status, _ = runtimeRequest(t, port, "runtime-tenant", token)
	require.Equal(t, http.StatusForbidden, status, "same-name recreation must not inherit installation assignment")
	assignRuntimeNamespace(t, installationUID)
	require.Eventually(t, func() bool {
		status, _ := runtimeRequest(t, port, "runtime-tenant", token)

		return status == http.StatusOK
	}, timeout, poll)
	_, recreated := runtimeRequest(t, port, "runtime-tenant", token)
	require.NotEqual(t, scope["namespaceUID"], recreated["namespaceUID"], "discovery must return the current UID")

	kubectl(t, "delete", "serviceaccount", "runtime-reader", "-n", "opamp-e2e")
	kubectl(t, "create", "serviceaccount", "runtime-reader", "-n", "opamp-e2e")
	require.Eventually(t, func() bool {
		status, _ := runtimeRequest(t, port, "runtime-tenant", token)

		return status == http.StatusUnauthorized
	}, timeout, poll, "recreating a ServiceAccount must not restore its old token")
}

func assignRuntimeNamespace(t *testing.T, installationUID string) {
	t.Helper()
	kubectl(t, "label", "namespace", "runtime-tenant",
		"opampcommander.io/installation-uid="+installationUID, "--overwrite")
}

func grantRuntimeReader(t *testing.T) {
	t.Helper()
	kubectl(t, "create", "role", "runtime-reader", "-n", "runtime-tenant",
		"--verb=get", "--resource=namespaces.runtime.opampcommander.io", "--resource-name=runtime-tenant")
	kubectl(t, "create", "rolebinding", "runtime-reader", "-n", "runtime-tenant",
		"--role=runtime-reader", "--serviceaccount=opamp-e2e:runtime-reader")
}

func runtimeToken(t *testing.T, audience string) string {
	t.Helper()

	return strings.TrimSpace(string(kubectl(t, "create", "token", "runtime-reader", "-n", "opamp-e2e",
		"--audience="+audience, "--duration=10m")))
}

func runtimeRequest(t *testing.T, port int, namespace, token string) (int, map[string]string) {
	t.Helper()

	url := fmt.Sprintf("http://127.0.0.1:%d/api/runtime/v1/namespaces/%s", port, namespace)
	//nolint:gosec // URL targets this test's loopback port-forward and fixed namespace fixtures.
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	require.NoError(t, err)

	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	request.Header.Set("X-Remote-User", "system:admin")
	request.Header.Set("X-Remote-Group", "system:masters")
	request.Header.Set("Impersonate-User", "system:admin")

	client := &http.Client{Timeout: 10 * time.Second}
	//nolint:gosec // Request only reaches this disposable cluster via loopback.
	response, err := client.Do(request)
	require.NoError(t, err)

	defer func() { require.NoError(t, response.Body.Close()) }()

	var scope map[string]string

	if response.StatusCode == http.StatusOK {
		require.NoError(t, json.NewDecoder(response.Body).Decode(&scope))
	}

	return response.StatusCode, scope
}

func assertRuntimeInstallationFenced(t *testing.T, port int) {
	t.Helper()

	status, _ := runtimeRequest(t, port, "runtime-tenant", runtimeToken(t, "opampcommander-runtime"))
	require.Equal(t, http.StatusServiceUnavailable, status, "gateway must reject the replaced installation anchor")
}
