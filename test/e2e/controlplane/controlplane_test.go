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

	"github.com/open-telemetry/opamp-go/protobufs"
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

	apiClient := client.New(fmt.Sprintf("http://127.0.0.1:%d", port))
	require.NoError(t, apiClient.Ping())

	t.Run("runtime namespace ownership", func(t *testing.T) { testRuntimeOwnership(t, port) })

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

				t.Run("Mongo write failure and recovery", func(t *testing.T) {
					defer mongoWriteFailure(t, false)

					mongoWriteFailure(t, true)

					errorCount := len(agent.ServerErrors())
					require.NoError(t, agent.ReportHealth(false, "write must fail"))
					require.Eventually(t, func() bool {
						for _, response := range agent.ServerErrors()[errorCount:] {
							if response.GetType() == protobufs.ServerErrorResponseType_ServerErrorResponseType_Unavailable &&
								response.GetErrorMessage() == "failed to persist agent state" {
								return true
							}
						}

						return false
					}, timeout, poll, "failed writes must return Unavailable")
					stored, err := apiClient.AgentService.GetAgent(t.Context(), "default", agent.UID)
					require.NoError(t, err)
					require.True(t, stored.Status.ComponentHealth.Healthy, "failed report must not enter the read cache")

					mongoWriteFailure(t, false)
					require.NoError(t, agent.ReportHealth(false, "storage recovered"))
					testutil.EventuallyAgent(t, apiClient, "default", agent.UID, func(a *v1.Agent) bool {
						return !a.Status.ComponentHealth.Healthy && a.Status.ComponentHealth.Status == "storage recovered"
					}, timeout, poll, "new reports must persist after Mongo recovers")
				})

				t.Run("Redis data loss and recovery", func(t *testing.T) {
					livenessKey := "opampcommander:agentliveness:" + agent.UID.String()

					require.Equal(t, "OK", strings.TrimSpace(string(kubectl(t, "exec", "-n", "opamp-e2e",
						"deployment/redis", "--", "redis-cli", "FLUSHDB"))))
					require.Eventually(t, func() bool {
						return strings.TrimSpace(string(kubectl(t, "exec", "-n", "opamp-e2e",
							"deployment/redis", "--", "redis-cli", "EXISTS", livenessKey))) == "1"
					}, timeout, poll, "live Agent must rebuild its Redis record")
					// List reads Mongo directly, so the local read cache cannot hide durable data loss.
					persisted, err := apiClient.AgentService.ListAgents(t.Context(), "default")
					require.NoError(t, err)

					var found bool

					for _, item := range persisted.Items {
						if item.Metadata.InstanceUID == agent.UID {
							found = true

							require.Equal(t, "storage recovered", item.Status.ComponentHealth.Status)
							require.Equal(t, body, item.Status.EffectiveConfig.ConfigMap.ConfigMap[key].Body)
						}
					}

					require.True(t, found, "Redis loss must preserve the Mongo Agent record")
					require.Equal(t, body, agent.EffectiveConfig()[key])
				})
			})
		}
	})

	t.Run("native configuration and dependency recovery", func(t *testing.T) {
		testNativeConfiguration(t, apiClient)
		assertRuntimeInstallationFenced(t, port)
	})
}

func testNativeConfiguration(t *testing.T, apiClient *client.Client) {
	t.Helper()

	kubectl(t, "apply", "-k", "../../../operator/config/samples")
	anchorName := string(kubectl(t, "get", "configmap", "-n", operatorNamespace,
		"-l", "app.kubernetes.io/component=installation", "-o", "jsonpath={.items[0].metadata.name}"))
	anchorUID := string(kubectl(t, "get", "configmap", anchorName, "-n", operatorNamespace,
		"-o", "jsonpath={.metadata.uid}"))
	require.NotEmpty(t, anchorUID)

	for _, assignment := range []string{"", "another-installation"} {
		if assignment != "" {
			kubectl(t, "label", "namespace", "default", "opampcommander.io/installation-uid="+assignment, "--overwrite")
		}

		require.Never(t, func() bool {
			group := kubectl(t, "get", "agentgroup", "collectors", "-o", "jsonpath={.status.conditions}")
			config := kubectl(t, "get", "remoteconfig", "otlp-debug", "-o", "jsonpath={.status.conditions}")

			return len(group) != 0 || len(config) != 0
		}, 5*time.Second, poll, "unassigned and foreign namespaces must remain untouched")
	}

	kubectl(t, "label", "namespace", "default", "opampcommander.io/installation-uid="+anchorUID, "--overwrite")
	waitCondition(t, "RuntimeScopeVerified", "True")
	waitCondition(t, "ReferencesFound", "True")
	waitCondition(t, "Ready", "Unknown")
	require.Eventually(t, func() bool {
		return string(kubectl(t, "get", "remoteconfig", "otlp-debug", "-o",
			"jsonpath={.status.conditions[?(@.type=='Ready')].status}")) == "Unknown"
	}, timeout, poll, "namespace assignment must enqueue pre-existing RemoteConfigs")
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
	kubectl(t, "rollout", "status", "deployment/"+operatorDeployment, "-n", operatorNamespace, "--timeout=120s")

	// A same-name replacement must not silently change a running controller's installation.
	kubectl(t, "delete", "configmap", anchorName, "-n", operatorNamespace)
	kubectl(t, "create", "configmap", anchorName, "-n", operatorNamespace)
	replacementUID := string(kubectl(t, "get", "configmap", anchorName, "-n", operatorNamespace,
		"-o", "jsonpath={.metadata.uid}"))
	require.NotEqual(t, anchorUID, replacementUID)
	kubectl(t, "wait", "pod", "-n", operatorNamespace, "-l", "control-plane=controller-manager",
		"--for=condition=Ready=false", "--timeout=90s")
	kubectl(t, "patch", "agentgroup", "collectors", "--type=merge", "-p", `{"spec":{"priority":42}}`)
	before = groupResourceVersion(t)
	require.Never(t, func() bool { return groupResourceVersion(t) != before }, 5*time.Second, poll,
		"an obsolete installation must not write CR status")
	require.NoError(t, apiClient.Ping(), "gateway remains available when Operator identity is invalid")
}

func mongoWriteFailure(t *testing.T, enabled bool) {
	t.Helper()

	mode := "off"
	if enabled {
		mode = "alwaysOn"
	}

	command := fmt.Sprintf(`const result = db.adminCommand({configureFailPoint: "failCommand", mode: %q,
		data: {failCommands: ["update"], errorCode: 121}}); if (!result.ok) throw new Error(JSON.stringify(result));`, mode)
	kubectl(t, "exec", "-n", "opamp-e2e", "deployment/mongodb", "--", "mongosh", "--quiet", "--eval", command)
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
