//go:build kind_e2e

package controlplane_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func testRuntimeEnrollment(t *testing.T, port int) {
	t.Helper()

	const (
		namespace   = "runtime-enrollment"
		instanceUID = "abc115aa-730b-4c1d-8638-e2b64eb27b34"
		path        = "/api/runtime/v1/namespaces/" + namespace + "/agent-enrollments/" + instanceUID
	)

	installationUID := string(kubectl(t, "get", "configmap", "-n", operatorNamespace,
		"opampcommander-operator-opamp-installation", "-o", "jsonpath={.metadata.uid}"))
	kubectl(t, "create", "namespace", namespace)
	kubectl(t, "label", "namespace", namespace, "opampcommander.io/installation-uid="+installationUID)
	kubectl(t, "create", "serviceaccount", "collector", "-n", namespace)

	namespaceUID := string(kubectl(t, "get", "namespace", namespace, "-o", "jsonpath={.metadata.uid}"))
	accountUID := string(kubectl(t, "-n", namespace, "get", "serviceaccount", "collector",
		"-o", "jsonpath={.metadata.uid}"))
	token := strings.TrimSpace(string(kubectl(t, "-n", namespace, "create", "token", "collector",
		"--audience=opampcommander-runtime", "--duration=10m")))

	scope := fmt.Sprintf(`{"namespaceUID":%q,"installationUID":%q}`, namespaceUID, installationUID)
	status, _ := runtimeRequestWithMethod(t, port, http.MethodPut, path, token, strings.NewReader(scope))
	require.Equal(t, http.StatusForbidden, status)

	kubectl(t, "apply", "-f", "runtime-enrollment.yaml")
	require.Eventually(t, func() bool {
		status, _ := runtimeRequestWithMethod(t, port, http.MethodGet, path, token, nil)

		return status == http.StatusNotFound
	}, timeout, poll, "authorized reads must distinguish a missing enrollment")

	require.Eventually(t, func() bool {
		status, _ := runtimeRequestWithMethod(t, port, http.MethodPut, path, token, strings.NewReader(scope))

		return status == http.StatusNoContent
	}, timeout, poll, "an authorized ServiceAccount must be able to enroll its Agent")

	status, _ = runtimeRequestWithMethod(t, port, http.MethodPut, path, token, strings.NewReader(scope))
	require.Equal(t, http.StatusNoContent, status, "registration retries must be idempotent")

	status, enrollment := runtimeRequestWithMethod(t, port, http.MethodGet, path, token, nil)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, map[string]string{
		"installationUID": installationUID, "namespaceUID": namespaceUID,
		"instanceUID": instanceUID, "serviceAccountUID": accountUID,
	}, enrollment)

	otherPath := strings.TrimSuffix(path, instanceUID) + "00000000-0000-0000-0000-000000000001"
	status, _ = runtimeRequestWithMethod(t, port, http.MethodGet, otherPath, token, nil)
	require.Equal(t, http.StatusForbidden, status, "RBAC permits only the named Agent")

	kubectl(t, "-n", namespace, "delete", "rolebinding", "agent-enroller")
	require.Eventually(t, func() bool {
		status, _ := runtimeRequestWithMethod(t, port, http.MethodGet, path, token, nil)

		return status == http.StatusForbidden
	}, timeout, poll, "lookup permission must be checked again after RBAC revocation")
}
