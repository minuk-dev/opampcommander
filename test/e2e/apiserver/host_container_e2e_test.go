//go:build e2e

package apiserver_test

import (
	"context"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

// waitForAgentDescription blocks until the agent identified by uid has reported a
// non-empty description, then returns it. It fails the test if that does not
// happen within the timeout (which would mean the collector never reported its
// attributes — a prerequisite for discovery).
func waitForAgentDescription(
	ctx context.Context,
	t *testing.T,
	c *client.Client,
	uid string,
) v1.AgentDescription {
	t.Helper()

	var desc v1.AgentDescription

	require.Eventually(t, func() bool {
		agents, err := c.AgentService.ListAgents(ctx, "default")
		if err != nil {
			return false
		}

		for _, agent := range agents.Items {
			if agent.Metadata.InstanceUID.String() != uid {
				continue
			}

			if len(agent.Metadata.Description.NonIdentifyingAttributes) == 0 {
				return false
			}

			desc = agent.Metadata.Description

			return true
		}

		return false
	}, 2*time.Minute, 2*time.Second, "agent should register and report its description")

	return desc
}

// TestE2E_Host_Discovery verifies that an agent reporting host.* attributes is
// discovered as a Host aggregate, classified by platform, and exposes its agent
// through the /hosts/{id}/agents sub-resource.
func TestE2E_Host_Discovery(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()

	base := testutil.NewBase(t)
	mongoServer := base.StartMongoDB()
	apiServer := base.StartAPIServer(mongoServer.URI, "opampcommander_e2e_host_discovery")
	defer apiServer.Stop()

	apiServer.WaitForReady()

	opampClient := apiServer.Client()

	const hostID = "e2e-host-1"

	// Given: a collector reporting host attributes plus a cloud provider (=> vm platform).
	collector := base.StartOTelCollectorWithDescription(
		apiServer.Port,
		map[string]string{
			"host.id":        hostID,
			"host.name":      "e2e-node-1",
			"host.arch":      "amd64",
			"cloud.provider": "aws",
		},
	)
	defer func() { _ = collector.Terminate(ctx) }()

	// Guard: the attributes must actually reach the agent description.
	desc := waitForAgentDescription(ctx, t, opampClient, collector.UID.String())
	require.Equal(t, hostID, desc.NonIdentifyingAttributes["host.id"],
		"collector should report host.id in its description")

	// When/Then: the host is discovered with the expected identity and platform.
	require.Eventually(t, func() bool {
		host, err := opampClient.HostService.GetHost(ctx, hostID)
		if err != nil {
			return false
		}

		return host.Spec.Platform == "vm" &&
			lo.Contains(host.Status.AgentInstanceUIDs, collector.UID.String())
	}, time.Minute, 2*time.Second, "host should be discovered with vm platform and the agent associated")

	// Then: the host appears in the list.
	hosts, err := opampClient.HostService.ListHosts(ctx)
	require.NoError(t, err)
	assert.True(t, lo.ContainsBy(hosts.Items, func(h v1.Host) bool {
		return h.Metadata.ID == hostID
	}), "discovered host should appear in the list")

	// Then: the host's agents sub-resource returns the collector's agent.
	agents, err := opampClient.HostService.ListAgentsByHost(ctx, hostID)
	require.NoError(t, err)
	assert.True(t, lo.ContainsBy(agents.Items, func(a v1.Agent) bool {
		return a.Metadata.InstanceUID == collector.UID
	}), "host agents sub-resource should include the collector")
}

// TestE2E_Container_Discovery verifies that an agent reporting container/k8s
// attributes is discovered as a Container aggregate keyed by pod uid, classified
// as kubernetes, and linked to its node host.
func TestE2E_Container_Discovery(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()

	base := testutil.NewBase(t)
	mongoServer := base.StartMongoDB()
	apiServer := base.StartAPIServer(mongoServer.URI, "opampcommander_e2e_container_discovery")
	defer apiServer.Stop()

	apiServer.WaitForReady()

	opampClient := apiServer.Client()

	const (
		podUID   = "e2e-pod-uid-1"
		nodeName = "e2e-node-7"
	)

	// Given: a collector reporting container + kubernetes attributes (=> kubernetes platform).
	collector := base.StartOTelCollectorWithDescription(
		apiServer.Port,
		map[string]string{
			"container.id":         "e2e-container-1",
			"container.runtime":    "containerd",
			"container.image.name": "otel/opentelemetry-collector-contrib",
			"k8s.pod.uid":          podUID,
			"k8s.pod.name":         "otelcol-e2e",
			"k8s.node.name":        nodeName,
		},
	)
	defer func() { _ = collector.Terminate(ctx) }()

	// Guard: the attributes must actually reach the agent description.
	desc := waitForAgentDescription(ctx, t, opampClient, collector.UID.String())
	require.Equal(t, podUID, desc.NonIdentifyingAttributes["k8s.pod.uid"],
		"collector should report k8s.pod.uid in its description")

	// When/Then: the container is discovered keyed by pod uid, classified kubernetes,
	// and linked to a host. (The collector also auto-reports its own host.name, so the
	// resolved HostID may be that rather than the node name; the exact node-name
	// fallback is covered by the domain unit tests. Here we assert the link exists.)
	require.Eventually(t, func() bool {
		container, err := opampClient.ContainerService.GetContainer(ctx, podUID)
		if err != nil {
			return false
		}

		return container.Spec.Platform == "kubernetes" &&
			container.Spec.HostID != "" &&
			lo.Contains(container.Status.AgentInstanceUIDs, collector.UID.String())
	}, time.Minute, 2*time.Second, "container should be discovered as kubernetes linked to a host")

	// Then: the container appears in the list.
	containers, err := opampClient.ContainerService.ListContainers(ctx)
	require.NoError(t, err)
	assert.True(t, lo.ContainsBy(containers.Items, func(c v1.Container) bool {
		return c.Metadata.ID == podUID
	}), "discovered container should appear in the list")

	// Then: the container's agents sub-resource returns the collector's agent.
	agents, err := opampClient.ContainerService.ListAgentsByContainer(ctx, podUID)
	require.NoError(t, err)
	assert.True(t, lo.ContainsBy(agents.Items, func(a v1.Agent) bool {
		return a.Metadata.InstanceUID == collector.UID
	}), "container agents sub-resource should include the collector")
}

// TestE2E_Host_ListAgents_Pagination verifies that the /hosts/{id}/agents
// sub-resource paginates: two collectors sharing one host produce two associated
// agents, and a limit=1 request returns one item plus a continue token that
// resumes the remaining page.
func TestE2E_Host_ListAgents_Pagination(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()

	base := testutil.NewBase(t)
	mongoServer := base.StartMongoDB()
	apiServer := base.StartAPIServer(mongoServer.URI, "opampcommander_e2e_host_pagination")
	defer apiServer.Stop()

	apiServer.WaitForReady()

	opampClient := apiServer.Client()

	const hostID = "e2e-shared-host"

	// Given: two collectors reporting the same host.id (=> two agents on one host).
	nonIdentifying := map[string]string{"host.id": hostID, "host.name": "e2e-shared-node"}
	collector1 := base.StartOTelCollectorWithDescription(apiServer.Port, nonIdentifying)
	defer func() { _ = collector1.Terminate(ctx) }()

	collector2 := base.StartOTelCollectorWithDescription(apiServer.Port, nonIdentifying)
	defer func() { _ = collector2.Terminate(ctx) }()

	// When/Then: both agents become associated with the shared host.
	require.Eventually(t, func() bool {
		agents, err := opampClient.HostService.ListAgentsByHost(ctx, hostID)
		if err != nil {
			return false
		}

		return len(agents.Items) >= 2
	}, 2*time.Minute, 2*time.Second, "both collectors should associate with the shared host")

	// Then: a limit=1 page returns one item and a continue token.
	firstPage, err := opampClient.HostService.ListAgentsByHost(ctx, hostID, client.WithLimit(1))
	require.NoError(t, err)
	require.Len(t, firstPage.Items, 1, "first page should hold exactly one agent")
	require.NotEmpty(t, firstPage.Metadata.Continue, "first page should carry a continue token")

	// Then: resuming from the continue token returns the next page.
	secondPage, err := opampClient.HostService.ListAgentsByHost(
		ctx, hostID,
		client.WithLimit(1),
		client.WithContinueToken(firstPage.Metadata.Continue),
	)
	require.NoError(t, err)
	require.Len(t, secondPage.Items, 1, "second page should hold exactly one agent")

	// Then: the two pages cover distinct agents.
	assert.NotEqual(t, firstPage.Items[0].Metadata.InstanceUID, secondPage.Items[0].Metadata.InstanceUID,
		"paginated pages should not repeat the same agent")
}
