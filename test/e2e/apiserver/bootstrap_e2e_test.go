//go:build e2e

package apiserver_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

// TestE2E_APIServer_BootstrapReconcile verifies the declarative bootstrap end-to-end
// against a persistent MongoDB:
//   - on first start the initial manifests (configs/apiserver/initial) seed the
//     built-in default namespace and role;
//   - the reconcile is full-overwrite, so a permission an admin adds to the built-in
//     default role via the API is reset back to the manifest list on the next start.
func TestE2E_APIServer_BootstrapReconcile(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	base := testutil.NewBase(t)
	mongo := base.StartMongoDB()

	const dbName = "opampcommander_e2e_bootstrap_test"

	// --- first start: embedded manifests seed the defaults ---
	srv1 := base.StartAPIServer(mongo.URI, dbName)
	srv1.WaitForReady()
	client1 := srv1.Client()

	defaultNS, err := client1.NamespaceService.GetNamespace(t.Context(), "default")
	require.NoError(t, err, "default namespace must be seeded from the initial manifests")
	assert.Equal(t, "default", defaultNS.Metadata.Name)

	defaultRole := findRoleByName(t, client1, "default")
	require.NotNil(t, defaultRole, "default role must be seeded on first start")
	assert.True(t, defaultRole.Spec.IsBuiltIn)
	assert.Contains(t, defaultRole.Spec.Permissions, "agent:GET")
	require.NotContains(t, defaultRole.Spec.Permissions, "certificate:GET")

	// --- admin adds an out-of-manifest permission to the built-in default role ---
	updated := *defaultRole
	updated.Spec.Permissions = append(
		append([]string(nil), defaultRole.Spec.Permissions...), "certificate:GET")

	_, err = client1.RoleService.UpdateRole(t.Context(), defaultRole.Metadata.UID, &updated)
	require.NoError(t, err)

	afterEdit := findRoleByName(t, client1, "default")
	require.Contains(t, afterEdit.Spec.Permissions, "certificate:GET",
		"the admin edit must be persisted before the restart")

	srv1.Stop()

	// --- second start against the SAME database: full-overwrite reconcile resets it ---
	srv2 := base.StartAPIServer(mongo.URI, dbName)
	defer srv2.Stop()

	srv2.WaitForReady()

	client2 := srv2.Client()

	reconciled := findRoleByName(t, client2, "default")
	require.NotNil(t, reconciled)
	assert.NotContains(t, reconciled.Spec.Permissions, "certificate:GET",
		"full-overwrite reconcile must reset admin-added permissions to the manifest list")
	assert.Contains(t, reconciled.Spec.Permissions, "agent:GET",
		"manifest permissions must remain after reconcile")
}

// findRoleByName returns the role whose displayName matches name, or nil.
func findRoleByName(t *testing.T, c *client.Client, name string) *v1.Role {
	t.Helper()

	resp, err := c.RoleService.ListRoles(t.Context())
	require.NoError(t, err)

	for i := range resp.Items {
		if resp.Items[i].Spec.DisplayName == name {
			return &resp.Items[i]
		}
	}

	return nil
}
