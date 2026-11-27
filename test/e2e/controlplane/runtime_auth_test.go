//go:build kind_e2e

package controlplane_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func testRuntimeOwnership(t *testing.T, port int) {
	t.Helper()

	installationUID := string(kubectl(t, "get", "configmap", "-n", operatorNamespace,
		"opampcommander-operator-opamp-installation", "-o", "jsonpath={.metadata.uid}"))
	kubectl(t, "create", "namespace", "runtime-tenant")
	status, _ := runtimeRequest(t, port, "runtime-tenant")
	require.Equal(t, http.StatusForbidden, status)

	kubectl(t, "label", "namespace", "runtime-tenant", "opampcommander.io/installation-uid="+installationUID)
	status, scope := runtimeRequest(t, port, "runtime-tenant")
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, installationUID, scope["installationUID"])
	require.Equal(t, string(kubectl(t, "get", "namespace", "runtime-tenant", "-o", "jsonpath={.metadata.uid}")),
		scope["namespaceUID"])

	kubectl(t, "delete", "namespace", "runtime-tenant", "--wait=true", "--timeout=90s")
	kubectl(t, "create", "namespace", "runtime-tenant")
	status, _ = runtimeRequest(t, port, "runtime-tenant")
	require.Equal(t, http.StatusForbidden, status)

	kubectl(t, "label", "namespace", "runtime-tenant", "opampcommander.io/installation-uid="+installationUID)
	status, recreated := runtimeRequest(t, port, "runtime-tenant")
	require.Equal(t, http.StatusOK, status)
	require.NotEqual(t, scope["namespaceUID"], recreated["namespaceUID"])
}

func runtimeRequest(t *testing.T, port int, namespace string) (int, map[string]string) {
	t.Helper()

	url := fmt.Sprintf("http://127.0.0.1:%d/api/runtime/v1/namespaces/%s", port, namespace)
	//nolint:gosec // URL targets this test's loopback port-forward and fixed namespace fixtures.
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	require.NoError(t, err)

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

	status, _ := runtimeRequest(t, port, "runtime-tenant")
	require.Equal(t, http.StatusServiceUnavailable, status, "gateway must reject the replaced installation anchor")
}
