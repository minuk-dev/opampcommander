//go:build e2e

package apiserver_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

// Server-side selectors are only worth having if they behave the same through the
// whole stack as they do in the adapter parity tests: the query parameters have to
// parse, reach the datastore, and come back with a paginated total that describes
// the set the page was drawn from. Anything applied after a page is cut looks
// identical in a unit test and wrong here.

// startSelectorAPIServer brings up an isolated MongoDB and apiserver.
func startSelectorAPIServer(t *testing.T, dbName string) *client.Client {
	t.Helper()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	base := testutil.NewBase(t)
	mongoServer := base.StartMongoDB()
	apiServer := base.StartAPIServer(mongoServer.URI, dbName)
	t.Cleanup(apiServer.Stop)

	apiServer.WaitForReady()

	return apiServer.Client()
}

// seedEndpoint creates one labelled endpoint.
func seedEndpoint(t *testing.T, cli *client.Client, name string, attributes map[string]string, protocol string) {
	t.Helper()

	//exhaustruct:ignore
	_, err := cli.EndpointService.CreateEndpoint(t.Context(), "default", &v1.Endpoint{
		Metadata: v1.EndpointMetadata{Name: name, Namespace: "default", Attributes: attributes},
		Spec:     v1.EndpointSpec{URL: "https://" + name + ".example.com", Protocol: protocol},
	})
	require.NoError(t, err)
}

// listEndpointNames lists endpoints under the given options and returns their
// names plus the count the server says is still to come.
func listEndpointNames(t *testing.T, cli *client.Client, opts ...client.ListOption) ([]string, int64) {
	t.Helper()

	resp, err := cli.EndpointService.ListEndpoints(t.Context(), "default", opts...)
	require.NoError(t, err)

	names := make([]string, 0, len(resp.Items))
	for _, item := range resp.Items {
		names = append(names, item.Metadata.Name)
	}

	return names, resp.Metadata.RemainingItemCount
}

// requireBadRequest asserts that err is a 400 and returns the response body, so a
// caller can assert on what the server said was wrong. Every selector the server
// cannot answer is a 400 — that is the contract these tests exist to pin.
func requireBadRequest(t *testing.T, err error) string {
	t.Helper()
	require.Error(t, err)

	var responseError *client.ResponseError

	require.ErrorAs(t, err, &responseError)
	require.Equal(t, http.StatusBadRequest, responseError.StatusCode)

	return responseError.ErrorMessage
}

func TestE2E_Selectors_LabelAndField(t *testing.T) {
	t.Parallel()

	cli := startSelectorAPIServer(t, "opampcommander_e2e_selectors")

	seedEndpoint(t, cli, "prod-metrics", map[string]string{"env": "prod", "tier": "web"}, "otlp")
	seedEndpoint(t, cli, "prod-traces", map[string]string{"env": "prod", "tier": "canary"}, "otlp")
	seedEndpoint(t, cli, "staging-metrics", map[string]string{"env": "staging", "tier": "web"}, "otlphttp")
	seedEndpoint(t, cli, "unlabelled", nil, "otlp")

	tests := map[string]struct {
		options  []client.ListOption
		expected []string
	}{
		"no selector returns everything": {
			nil,
			[]string{"prod-metrics", "prod-traces", "staging-metrics", "unlabelled"},
		},
		"label equality": {
			[]client.ListOption{client.WithLabelSelector("env=prod")},
			[]string{"prod-metrics", "prod-traces"},
		},
		"label inequality also selects the resource carrying no such label": {
			[]client.ListOption{client.WithLabelSelector("env!=prod")},
			[]string{"staging-metrics", "unlabelled"},
		},
		"set membership": {
			[]client.ListOption{client.WithLabelSelector("tier in (web,canary)")},
			[]string{"prod-metrics", "prod-traces", "staging-metrics"},
		},
		"existence": {
			[]client.ListOption{client.WithLabelSelector("tier")},
			[]string{"prod-metrics", "prod-traces", "staging-metrics"},
		},
		"non-existence": {
			[]client.ListOption{client.WithLabelSelector("!tier")},
			[]string{"unlabelled"},
		},
		"conjunction": {
			[]client.ListOption{client.WithLabelSelector("env=prod,tier!=canary")},
			[]string{"prod-metrics"},
		},
		"field selector": {
			[]client.ListOption{client.WithFieldSelector("spec.protocol=otlphttp")},
			[]string{"staging-metrics"},
		},
		"label and field selectors combine": {
			[]client.ListOption{
				client.WithLabelSelector("tier=web"),
				client.WithFieldSelector("spec.protocol=otlp"),
			},
			[]string{"prod-metrics"},
		},
		"name prefix": {
			[]client.ListOption{client.WithName("prod-")},
			[]string{"prod-metrics", "prod-traces"},
		},
		"name substring reaches inside the name, case-insensitively": {
			[]client.ListOption{client.WithNameContains("METRICS")},
			[]string{"prod-metrics", "staging-metrics"},
		},
		"a name substring is matched literally, never as a pattern": {
			[]client.ListOption{client.WithNameContains("prod-.*")},
			nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			names, _ := listEndpointNames(t, cli, test.options...)
			assert.ElementsMatch(t, test.expected, names)
		})
	}
}

// The property the whole design is built around: a filter reaches the datastore
// query, so the page and its "how many are left" describe the same set. A filter
// applied to an already-cut page would report two remaining here.
func TestE2E_Selectors_PaginationCountsOnlyMatchingItems(t *testing.T) {
	t.Parallel()

	cli := startSelectorAPIServer(t, "opampcommander_e2e_selectors_paging")

	seedEndpoint(t, cli, "prod-metrics", map[string]string{"env": "prod"}, "otlp")
	seedEndpoint(t, cli, "prod-traces", map[string]string{"env": "prod"}, "otlp")
	seedEndpoint(t, cli, "staging-metrics", map[string]string{"env": "staging"}, "otlp")
	seedEndpoint(t, cli, "staging-traces", map[string]string{"env": "staging"}, "otlp")

	names, remaining := listEndpointNames(t, cli,
		client.WithLabelSelector("env=prod"), client.WithLimit(1))

	require.Len(t, names, 1)
	assert.Equal(t, int64(1), remaining,
		"two endpoints match env=prod, so a page of one leaves exactly one behind")
}

// A selector the server cannot answer is a 400 naming what was wrong, never a
// 200 carrying the whole collection: a client that asked to narrow a list must
// not be able to mistake the unfiltered one for a filtered one.
func TestE2E_Selectors_RejectedRatherThanIgnored(t *testing.T) {
	t.Parallel()

	cli := startSelectorAPIServer(t, "opampcommander_e2e_selectors_rejected")

	seedEndpoint(t, cli, "prod-metrics", map[string]string{"env": "prod"}, "otlp")

	t.Run("an unsupported field names itself and the supported ones", func(t *testing.T) {
		t.Parallel()

		_, err := cli.EndpointService.ListEndpoints(t.Context(), "default",
			client.WithFieldSelector("spec.nope=1"))

		body := requireBadRequest(t, err)
		assert.Contains(t, body, "spec.nope")
		assert.Contains(t, body, "spec.protocol")
	})

	t.Run("a malformed label selector is rejected", func(t *testing.T) {
		t.Parallel()

		_, err := cli.EndpointService.ListEndpoints(t.Context(), "default",
			client.WithLabelSelector("env in prod"))

		_ = requireBadRequest(t, err)
	})

	t.Run("a label selector on a resource with no metadata says so", func(t *testing.T) {
		t.Parallel()

		_, err := cli.RoleService.ListRoles(t.Context(), client.WithLabelSelector("env=prod"))

		assert.Contains(t, requireBadRequest(t, err), "no labels or attributes")
	})

	// The two metadata selectors are not interchangeable, and sending the wrong
	// one must not be an ignored parameter: gin drops what nothing reads, so the
	// caller would get the whole collection with a 200.
	t.Run("a label selector on agents names the parameter agents do have", func(t *testing.T) {
		t.Parallel()

		_, err := cli.AgentService.ListAgents(t.Context(), "default",
			client.WithLabelSelector("os.type=linux"))

		body := requireBadRequest(t, err)
		assert.Contains(t, body, "labelSelector")
		assert.Contains(t, body, "attributeSelector")
	})

	t.Run("an attribute selector on a labelled resource names labelSelector", func(t *testing.T) {
		t.Parallel()

		_, err := cli.EndpointService.ListEndpoints(t.Context(), "default",
			client.WithAttributeSelector("env=prod"))

		body := requireBadRequest(t, err)
		assert.Contains(t, body, "attributeSelector")
		assert.Contains(t, body, "labelSelector")
	})
}

// Roles have no label map but do have a name and a field, which is what makes
// them worth filtering at all.
func TestE2E_Selectors_RolesFilterWithoutLabels(t *testing.T) {
	t.Parallel()

	cli := startSelectorAPIServer(t, "opampcommander_e2e_selectors_roles")

	builtIn, err := cli.RoleService.ListRoles(t.Context(),
		client.WithFieldSelector("spec.isBuiltIn=true"))
	require.NoError(t, err)
	require.NotEmpty(t, builtIn.Items, "the bootstrap seeds built-in roles")

	for _, role := range builtIn.Items {
		assert.True(t, role.Spec.IsBuiltIn)
	}

	custom, err := cli.RoleService.ListRoles(t.Context(),
		client.WithFieldSelector("spec.isBuiltIn=false"))
	require.NoError(t, err)

	for _, role := range custom.Items {
		assert.False(t, role.Spec.IsBuiltIn)
	}
}

// A namespaced listing must honour the namespace in its own path. Authorization
// is enforced per-namespace, so a listing that ignored it would hand a caller
// authorized for one namespace the contents of every other.
func TestE2E_RoleBindings_ScopedToTheirNamespace(t *testing.T) {
	t.Parallel()

	cli := startSelectorAPIServer(t, "opampcommander_e2e_rolebinding_scope")

	role, err := cli.RoleService.CreateRole(t.Context(), &v1.Role{
		Kind:       v1.RoleKind,
		APIVersion: v1.APIVersion,
		//exhaustruct:ignore
		Metadata: v1.RoleMetadata{},
		Spec: v1.RoleSpec{
			DisplayName: "Scope Test Role",
			Description: "role bound in two namespaces",
			Permissions: []string{"agent:GET"},
			IsBuiltIn:   false,
		},
		//exhaustruct:ignore
		Status: v1.RoleStatus{},
	})
	require.NoError(t, err)

	other := "scope-other"

	//exhaustruct:ignore
	_, err = cli.NamespaceService.CreateNamespace(t.Context(), &v1.Namespace{
		Metadata: v1.NamespaceMetadata{Name: other},
	})
	require.NoError(t, err)

	for _, namespace := range []string{"default", other} {
		//exhaustruct:ignore
		_, err = cli.RoleBindingService.CreateRoleBinding(t.Context(), namespace, &v1.RoleBinding{
			Kind:       v1.RoleBindingKind,
			APIVersion: v1.APIVersion,
			Metadata:   v1.RoleBindingMetadata{Namespace: namespace, Name: "scope-binding"},
			Spec: v1.RoleBindingSpec{
				RoleRef:  v1.RoleBindingRoleRef{Kind: "Role", Name: role.Spec.DisplayName},
				Subjects: []v1.RoleBindingSubject{{Kind: "User", Name: "test@test.com"}},
			},
		})
		require.NoError(t, err)
	}

	for _, namespace := range []string{"default", other} {
		resp, err := cli.RoleBindingService.ListRoleBindings(t.Context(), namespace)
		require.NoError(t, err)

		for _, binding := range resp.Items {
			assert.Equal(t, namespace, binding.Metadata.Namespace,
				"a namespaced listing must not return another namespace's bindings")
		}
	}
}
