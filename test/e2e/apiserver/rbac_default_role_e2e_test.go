//go:build e2e

package apiserver_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

//nolint:funlen,cyclop // E2E test function with many sequential test steps.
func TestE2E_APIServer_DefaultRole(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()

	const dbName = "opampcommander_e2e_default_role_test"

	base := testutil.NewBase(t)
	mongoServer := base.StartMongoDB()
	apiServer := base.StartAPIServer(mongoServer.URI, dbName)
	defer apiServer.Stop()

	apiServer.WaitForReady()

	opampClient := apiServer.Client()

	// Seed agent:GET and agent:LIST permissions for default role assignment.
	agentGetPerm := newPermissionSeed("agent", "GET")
	agentListPerm := newPermissionSeed("agent", "LIST")
	seedPermissions(t, ctx, mongoServer.URI, dbName, []permissionSeed{agentGetPerm, agentListPerm})

	// Assign seeded permissions to the default role.
	t.Run("Setup_AssignPermissionsToDefaultRole", func(t *testing.T) {
		roles, err := opampClient.RoleService.ListRoles(t.Context())
		require.NoError(t, err)

		var defaultRoleUID string

		for _, role := range roles.Items {
			if role.Spec.DisplayName == "default" {
				defaultRoleUID = role.Metadata.UID

				break
			}
		}

		require.NotEmpty(t, defaultRoleUID, "default role must exist after server start")

		_, err = opampClient.RoleService.UpdateRole(t.Context(), defaultRoleUID, &v1.Role{
			Kind:       v1.RoleKind,
			APIVersion: v1.APIVersion,
			//exhaustruct:ignore
			Metadata: v1.RoleMetadata{UID: defaultRoleUID},
			Spec: v1.RoleSpec{
				DisplayName: "default",
				Description: "Default role assigned to all new users on first login",
				Permissions: []string{agentGetPerm.name, agentListPerm.name},
				IsBuiltIn:   true,
			},
			//exhaustruct:ignore
			Status: v1.RoleStatus{},
		})
		require.NoError(t, err, "failed to update default role with permissions")
	})

	// Create a new user with no explicit RoleBindings.
	newUser, err := opampClient.UserService.CreateUser(t.Context(), &v1.User{
		Kind:       v1.UserKind,
		APIVersion: v1.APIVersion,
		//exhaustruct:ignore
		Metadata: v1.UserMetadata{
			Labels: map[string]string{"test-group": "default-role-test"},
		},
		Spec: v1.UserSpec{Email: "newuser@example.com", Username: "new-user", IsActive: true},
		//exhaustruct:ignore
		Status: v1.UserStatus{},
	})
	require.NoError(t, err, "failed to create new user")

	newUserUID := newUser.Metadata.UID
	require.NoError(t, opampClient.RBACService.SyncPolicies(t.Context()))

	// =================================================================
	// Phase 1: Verify new user has exactly the default role (and no explicit bindings)
	// =================================================================
	t.Run("Phase1_NewUserHasOnlyDefaultRole", func(t *testing.T) {
		roles, err := opampClient.RBACService.GetUserRoles(t.Context(), newUserUID)
		require.NoError(t, err)

		roleNames := make([]string, 0, len(roles.Items))
		for _, role := range roles.Items {
			roleNames = append(roleNames, role.Spec.DisplayName)
		}

		assert.Len(t, roles.Items, 1, "new user should have exactly one role (default)")
		assert.Contains(t, roleNames, "default", "new user must have the default role")
	})

	t.Run("Phase1_NewUserHasNoRoleBindings", func(t *testing.T) {
		// Admin lists all role bindings and verifies none match the new user's label.
		allBindings, err := opampClient.RoleBindingService.ListRoleBindings(t.Context())
		require.NoError(t, err)

		for _, binding := range allBindings.Items {
			selectorVal := binding.Spec.LabelSelector["test-group"]
			assert.NotEqual(t, "default-role-test", selectorVal,
				"no role binding should explicitly target the new user")
		}
	})

	// =================================================================
	// Phase 2: Verify default role grants GET/LIST in "default" namespace
	// =================================================================
	t.Run("Phase2_DefaultRoleAllowsGETInDefaultNamespace", func(t *testing.T) {
		perm, err := opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: newUserUID, Namespace: "default", Resource: "agent", Action: "GET",
		})
		require.NoError(t, err)
		assert.True(t, perm.Allowed, "expected ALLOWED: default/agent/GET")

		perm, err = opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: newUserUID, Namespace: "default", Resource: "agent", Action: "LIST",
		})
		require.NoError(t, err)
		assert.True(t, perm.Allowed, "expected ALLOWED: default/agent/LIST")
	})

	t.Run("Phase2_DefaultRoleDeniesWriteInDefaultNamespace", func(t *testing.T) {
		perm, err := opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: newUserUID, Namespace: "default", Resource: "agent", Action: "CREATE",
		})
		require.NoError(t, err)
		assert.False(t, perm.Allowed, "expected DENIED: default/agent/CREATE")

		perm, err = opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: newUserUID, Namespace: "default", Resource: "agent", Action: "DELETE",
		})
		require.NoError(t, err)
		assert.False(t, perm.Allowed, "expected DENIED: default/agent/DELETE")
	})

	t.Run("Phase2_DefaultRoleDoesNotApplyToOtherNamespaces", func(t *testing.T) {
		perm, err := opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: newUserUID, Namespace: "production", Resource: "agent", Action: "GET",
		})
		require.NoError(t, err)
		assert.False(t, perm.Allowed, "expected DENIED: production/agent/GET (default role is default-ns only)")

		perm, err = opampClient.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
			UserID: newUserUID, Namespace: "staging", Resource: "agent", Action: "LIST",
		})
		require.NoError(t, err)
		assert.False(t, perm.Allowed, "expected DENIED: staging/agent/LIST (default role is default-ns only)")
	})

	// =================================================================
	// Phase 3: Admin uses /me/roles and /me/rolebindings endpoints
	// =================================================================
	t.Run("Phase3_AdminCanQueryOwnRolesViaMe", func(t *testing.T) {
		// Create a role binding that targets the admin user (login-type: basic).
		_, err := opampClient.RoleBindingService.CreateRoleBinding(t.Context(), "default", &v1.RoleBinding{
			Kind:       v1.RoleBindingKind,
			APIVersion: v1.APIVersion,
			Metadata: v1.RoleBindingMetadata{
				Namespace: "default",
				Name:      "admin-agent-viewer",
				//exhaustruct:ignore
				CreatedAt: v1.Time{},
				//exhaustruct:ignore
				UpdatedAt: v1.Time{},
			},
			Spec: v1.RoleBindingSpec{
				RoleRef:       v1.RoleBindingRoleRef{Kind: "Role", Name: "default"},
				LabelSelector: map[string]string{"login-type": "basic"},
			},
			//exhaustruct:ignore
			Status: v1.RoleBindingStatus{},
		})
		require.NoError(t, err)

		myRoles, err := opampClient.UserService.GetMyRoles(t.Context())
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(myRoles.Items), 1, "admin should have at least the default role")

		roleNames := make([]string, 0, len(myRoles.Items))
		for _, role := range myRoles.Items {
			roleNames = append(roleNames, role.Spec.DisplayName)
		}

		assert.Contains(t, roleNames, "default", "/me/roles must include the default role")
	})

	t.Run("Phase3_AdminCanQueryOwnRoleBindingsViaMe", func(t *testing.T) {
		myBindings, err := opampClient.UserService.GetMyRoleBindings(t.Context())
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(myBindings.Items), 1,
			"admin should have at least the binding created above")

		found := false

		for _, binding := range myBindings.Items {
			if binding.Metadata.Name == "admin-agent-viewer" {
				found = true

				break
			}
		}

		assert.True(t, found, "admin's /me/rolebindings must include 'admin-agent-viewer'")
	})
}
