package integration_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

// TestStandaloneAPIServer exercises the full HTTP -> application -> domain ->
// in-memory persistence stack of an apiserver started in standalone mode
// (database type "inmemory", no MongoDB, no Kafka). It verifies that resources
// created over the API are stored and read back from the in-memory adapter, and
// that soft-deletes hide resources from default reads.
func TestStandaloneAPIServer(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	server := base.StartStandaloneAPIServer()

	t.Cleanup(server.Stop)
	server.WaitForReady()

	ctx := t.Context()
	apiClient := server.Client()

	t.Run("ping and version respond", func(t *testing.T) {
		t.Parallel()

		require.NoError(t, apiClient.Ping())

		version, err := apiClient.GetServerVersion(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, version.GoVersion)
	})

	t.Run("default namespace is seeded into the in-memory store", func(t *testing.T) {
		t.Parallel()

		def, err := apiClient.NamespaceService.GetNamespace(ctx, "default")
		require.NoError(t, err)
		assert.Equal(t, "default", def.Metadata.Name)
	})

	t.Run("namespace create, get, list round-trips through the in-memory store", func(t *testing.T) {
		t.Parallel()

		const name = "integration-ns"

		//exhaustruct:ignore
		created, err := apiClient.NamespaceService.CreateNamespace(ctx, &v1.Namespace{
			//exhaustruct:ignore
			Metadata: v1.NamespaceMetadata{Name: name},
		})
		require.NoError(t, err)
		assert.Equal(t, name, created.Metadata.Name)

		got, err := apiClient.NamespaceService.GetNamespace(ctx, name)
		require.NoError(t, err)
		assert.Equal(t, name, got.Metadata.Name)

		list, err := apiClient.NamespaceService.ListNamespaces(ctx)
		require.NoError(t, err)
		assert.True(t, containsNamespace(list.Items, name),
			"created namespace should appear in the listing")
	})

	t.Run("agent group lifecycle including soft-delete", func(t *testing.T) {
		t.Parallel()

		const groupName = "integration-grp"

		created, err := apiClient.AgentGroupService.CreateAgentGroup(ctx, "default", &v1.AgentGroup{
			//exhaustruct:ignore
			Metadata: v1.Metadata{Name: groupName},
			//exhaustruct:ignore
			Spec: v1.Spec{
				Priority: 7,
				Selector: v1.AgentSelector{
					IdentifyingAttributes: map[string]string{"service.name": "otelcol"},
				},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, groupName, created.Metadata.Name)
		// No agents exist, so the recomputed statistics are all zero.
		assert.Equal(t, 0, created.Status.NumAgents)

		got, err := apiClient.AgentGroupService.GetAgentGroup(ctx, "default", groupName)
		require.NoError(t, err)
		assert.Equal(t, 7, got.Spec.Priority)

		list, err := apiClient.AgentGroupService.ListAgentGroups(ctx, "default")
		require.NoError(t, err)
		assert.True(t, containsAgentGroup(list.Items, groupName),
			"created agent group should appear in the listing")

		// Soft-delete, then confirm it is hidden from the default read path.
		require.NoError(t, apiClient.AgentGroupService.DeleteAgentGroup(ctx, "default", groupName))

		_, err = apiClient.AgentGroupService.GetAgentGroup(ctx, "default", groupName)
		require.Error(t, err, "soft-deleted agent group should not be returned by default")

		listAfter, err := apiClient.AgentGroupService.ListAgentGroups(ctx, "default")
		require.NoError(t, err)
		assert.False(t, containsAgentGroup(listAfter.Items, groupName),
			"soft-deleted agent group should not appear in the default listing")
	})
}

// TestStandaloneAPIServer_Pagination verifies the in-memory store's cursor
// pagination over the public API: a limited list returns a continue token that
// resumes the listing without overlap.
func TestStandaloneAPIServer_Pagination(t *testing.T) {
	t.Parallel()

	base := testutil.NewBase(t)
	server := base.StartStandaloneAPIServer()

	t.Cleanup(server.Stop)
	server.WaitForReady()

	ctx := t.Context()
	apiClient := server.Client()

	const total = 5
	for _, name := range agentGroupNames(total) {
		_, err := apiClient.AgentGroupService.CreateAgentGroup(ctx, "default", &v1.AgentGroup{
			//exhaustruct:ignore
			Metadata: v1.Metadata{Name: name},
			//exhaustruct:ignore
			Spec: v1.Spec{
				Selector: v1.AgentSelector{
					IdentifyingAttributes: map[string]string{"service.name": name},
				},
			},
		})
		require.NoError(t, err)
	}

	seen := map[string]struct{}{}

	const pageSize = 2

	page, err := apiClient.AgentGroupService.ListAgentGroups(ctx, "default", client.WithLimit(pageSize))
	require.NoError(t, err)

	for {
		for _, group := range page.Items {
			_, dup := seen[group.Metadata.Name]
			require.False(t, dup, "pagination must not return the same group twice: %s", group.Metadata.Name)
			seen[group.Metadata.Name] = struct{}{}
		}

		if page.Metadata.Continue == "" {
			break
		}

		page, err = apiClient.AgentGroupService.ListAgentGroups(
			ctx, "default",
			client.WithLimit(pageSize),
			client.WithContinueToken(page.Metadata.Continue),
		)
		require.NoError(t, err)
	}

	assert.Len(t, seen, total, "every created agent group should be visited exactly once")
}

func agentGroupNames(n int) []string {
	names := make([]string, 0, n)
	for i := range n {
		names = append(names, "page-grp-"+string(rune('a'+i)))
	}

	return names
}

func containsNamespace(items []v1.Namespace, name string) bool {
	for i := range items {
		if items[i].Metadata.Name == name {
			return true
		}
	}

	return false
}

func containsAgentGroup(items []v1.AgentGroup, name string) bool {
	for i := range items {
		if items[i].Metadata.Name == name {
			return true
		}
	}

	return false
}
