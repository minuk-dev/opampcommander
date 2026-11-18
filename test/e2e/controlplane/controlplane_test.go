//go:build kind_e2e

package controlplane_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

const (
	operatorNamespace  = "opampcommander-operator-system"
	operatorDeployment = "opampcommander-operator-controller-manager"
	timeout            = 90 * time.Second
	poll               = time.Second
)

//nolint:paralleltest // These scenarios deliberately mutate one disposable cluster in sequence.
func TestControlPlane(t *testing.T) {
	cluster := os.Getenv("OPAMP_E2E_CLUSTER")
	require.True(t, strings.HasPrefix(cluster, "opamp-e2e-"), "run through make test-e2e-controlplane")
	require.NotEmpty(t, os.Getenv("KUBECONFIG"))
	require.Equal(t, "false", os.Getenv("KUBECTL_KUBERC"))
	require.Equal(t, "kind-"+cluster, strings.TrimSpace(string(kubectl(t, "config", "current-context"))))
	require.Equal(t, "kind-"+cluster, string(kubectl(t, "config", "view", "--minify",
		"-o", "jsonpath={.contexts[0].context.cluster}")))

	port, err := strconv.Atoi(os.Getenv("OPAMP_E2E_PORT"))
	require.NoError(t, err)

	apiClient := client.New(fmt.Sprintf("http://127.0.0.1:%d", port),
		client.WithBasicAuth(testutil.DefaultAdminUsername, testutil.DefaultAdminPassword))
	require.NoError(t, apiClient.Ping())

	t.Run("runtime wire contracts", func(t *testing.T) {
		for _, transport := range []testutil.ReferenceAgentTransport{
			testutil.ReferenceAgentTransportWebSocket, testutil.ReferenceAgentTransportHTTP,
		} {
			t.Run(string(transport), func(t *testing.T) {
				serviceName := "kind-" + string(transport)
				agent := testutil.NewBase(t).StartReferenceAgent(port,
					testutil.WithReferenceAgentTransport(transport),
					testutil.WithReferenceAgentIdentifyingAttributes(map[string]string{"service.name": serviceName}))
				testutil.EventuallyAgent(t, apiClient, "default", agent.UID, func(a *v1.Agent) bool {
					return a.Status.ComponentHealth.Healthy && a.Status.SequenceNum > 0
				}, timeout, poll, "registered Agent must report health and heartbeat")

				name, body := "collector.yaml", "service: {}\n# "+serviceName
				groupName := serviceName + "-config"
				_, err := apiClient.AgentGroupService.CreateAgentGroup(t.Context(), "default", &v1.AgentGroup{
					Metadata: v1.Metadata{Name: groupName},
					Spec: v1.Spec{
						Selector: v1.AgentSelector{IdentifyingAttributes: map[string]string{"service.name": serviceName}},
						AgentConfig: &v1.AgentConfig{AgentRemoteConfigs: []v1.AgentGroupRemoteConfig{{
							AgentRemoteConfigName: &name,
							AgentRemoteConfigSpec: &v1.AgentRemoteConfigSpec{Value: body, ContentType: "text/yaml"},
						}}},
					},
				})
				require.NoError(t, err)

				key := groupName + "/" + name

				require.Eventually(t, func() bool { return agent.EffectiveConfig()[key] == body }, timeout, poll)
				testutil.EventuallyAgent(t, apiClient, "default", agent.UID, func(a *v1.Agent) bool {
					return a.Status.EffectiveConfig.ConfigMap.ConfigMap[key].Body == body
				}, timeout, poll, "effective config must be reported back")
				_, err = apiClient.AgentService.RestartAgent(t.Context(), "default", agent.UID)
				require.NoError(t, err)
				require.Eventually(t, func() bool { return agent.Restarts() == 1 }, timeout, poll)
			})
		}
	})

	t.Run("native configuration and dependency recovery", func(t *testing.T) {
		kubectl(t, "apply", "-k", "../../../operator/config/samples")
		waitCondition(t, "ReferencesFound", "True")
		waitCondition(t, "Ready", "Unknown")
		before := groupResourceVersion(t)
		kubectl(t, "apply", "-k", "../../../operator/config/samples")
		require.Equal(t, before, groupResourceVersion(t), "no-op apply must not rewrite status")
		kubectl(t, "delete", "remoteconfig", "otlp-debug")
		waitCondition(t, "ReferencesFound", "False")
		kubectl(t, "apply", "-f", "../../../operator/config/samples/opampcommander_v1alpha1_remoteconfig.yaml")
		waitCondition(t, "ReferencesFound", "True")

		kubectl(t, "scale", "deployment/"+operatorDeployment, "-n", operatorNamespace, "--replicas=0")
		kubectl(t, "wait", "pod", "-n", operatorNamespace, "-l", "control-plane=controller-manager",
			"--for=delete", "--timeout=90s")
		kubectl(t, "delete", "remoteconfig", "otlp-debug")
		kubectl(t, "scale", "deployment/"+operatorDeployment, "-n", operatorNamespace, "--replicas=2")
		kubectl(t, "rollout", "status", "deployment/"+operatorDeployment, "-n", operatorNamespace, "--timeout=120s")
		waitCondition(t, "ReferencesFound", "False")
		kubectl(t, "apply", "-f", "../../../operator/config/samples/opampcommander_v1alpha1_remoteconfig.yaml")
		waitCondition(t, "ReferencesFound", "True")

		leader := string(kubectl(t, "get", "lease", "37ad389b.io", "-n", operatorNamespace,
			"-o", "jsonpath={.spec.holderIdentity}"))
		require.NotEmpty(t, leader)
		pod, _, _ := strings.Cut(leader, "_")
		kubectl(t, "delete", "pod", pod, "-n", operatorNamespace)
		require.Eventually(t, func() bool {
			current := string(kubectl(t, "get", "lease", "37ad389b.io", "-n", operatorNamespace,
				"-o", "jsonpath={.spec.holderIdentity}"))

			return current != "" && current != leader
		}, timeout, poll, "another Operator must acquire the Lease")
		kubectl(t, "delete", "remoteconfig", "otlp-debug")
		waitCondition(t, "ReferencesFound", "False")
		require.NoError(t, apiClient.Ping(), "gateway remains available during Operator failover")
	})
}

func kubectl(t *testing.T, args ...string) []byte {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 150*time.Second)
	defer cancel()

	//nolint:gosec // Arguments come from this test; no shell is involved.
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "kubectl %v: %s", args, output)

	return output
}

func waitCondition(t *testing.T, kind, status string) {
	t.Helper()
	require.Eventually(t, func() bool {
		query := fmt.Sprintf("jsonpath={.status.conditions[?(@.type=='%s')].status}", kind)

		return string(kubectl(t, "get", "agentgroup", "collectors", "-o", query)) == status
	}, timeout, poll, "%s must become %s", kind, status)
}

func groupResourceVersion(t *testing.T) string {
	t.Helper()

	return string(kubectl(t, "get", "agentgroup", "collectors", "-o", "jsonpath={.metadata.resourceVersion}"))
}
