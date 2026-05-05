//go:build e2e

package apiserver_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestE2E_APIServer_RBAC(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	base := testutil.NewBase(t)

	mongoServer := base.StartMongoDB()
	apiServer := base.StartAPIServer(mongoServer.URI, "opampcommander_e2e_rbac_test")
	defer apiServer.Stop()

	apiServer.WaitForReady()

	opampClient := apiServer.Client()

	// Test 1: Get current user profile via /api/v1/users/me
	t.Run("GetCurrentUserProfile", func(t *testing.T) {
		profile, err := opampClient.UserService.GetMyProfile(t.Context())
		require.NoError(t, err)
		assert.NotEmpty(t, profile.User.Metadata.UID)
		assert.Equal(t, "test@test.com", profile.User.Spec.Email)
		assert.Equal(t, "test-admin", profile.User.Spec.Username)
		assert.True(t, profile.User.Spec.IsActive)
	})

	// DefaultRole_HasBuiltInGetPermissions verifies that the built-in "default" role exists
	// at startup with GET permissions on every namespace-scoped resource. Together with the
	// SyncPolicies hook that auto-grants the default role to every user in the "default"
	// namespace, this is what gives every authenticated user read access there.
	t.Run("DefaultRole_HasBuiltInGetPermissions", func(t *testing.T) {
		expected := []string{
			"agent:GET",
			"agentgroup:GET",
			"agentpackage:GET",
			"agentremoteconfig:GET",
			"certificate:GET",
			"rolebinding:GET",
		}

		resp, err := opampClient.RoleService.ListRoles(t.Context())
		require.NoError(t, err)

		var defaultRole *v1.Role

		for i := range resp.Items {
			if resp.Items[i].Spec.DisplayName == "default" {
				defaultRole = &resp.Items[i]

				break
			}
		}

		require.NotNil(t, defaultRole, "built-in default role must exist")
		assert.True(t, defaultRole.Spec.IsBuiltIn, "default role must be marked built-in")

		for _, name := range expected {
			assert.True(t, slices.Contains(defaultRole.Spec.Permissions, name),
				"default role missing built-in permission %q (got %v)",
				name, defaultRole.Spec.Permissions)
		}
	})

	// DefaultRole_AppearsOnUserProfile verifies the default role surfaces on /users/me
	// for every authenticated user, with no RoleBinding (granted implicitly by SyncPolicies).
	t.Run("DefaultRole_AppearsOnUserProfile", func(t *testing.T) {
		profile, err := opampClient.UserService.GetMyProfile(t.Context())
		require.NoError(t, err)

		var defaultEntry *v1.UserRoleEntry

		for i := range profile.Roles {
			if profile.Roles[i].Role.Spec.DisplayName == "default" {
				defaultEntry = &profile.Roles[i]

				break
			}
		}

		require.NotNil(t, defaultEntry, "profile must include the default role")
		assert.Nil(t, defaultEntry.RoleBinding,
			"default role is granted implicitly — no RoleBinding should be reported")
		assert.NotEmpty(t, defaultEntry.Role.Spec.Permissions,
			"default role on profile must carry its built-in permissions")
	})

	// Test 2: List users
	t.Run("ListUsers", func(t *testing.T) {
		resp, err := opampClient.UserService.ListUsers(t.Context())
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(resp.Items), 1, "At least the admin user should exist")
	})

	// Test 3: Create a role
	var createdRoleUID string

	t.Run("CreateRole", func(t *testing.T) {
		role, err := opampClient.RoleService.CreateRole(t.Context(), &v1.Role{
			Kind:       v1.RoleKind,
			APIVersion: "v1",
			//exhaustruct:ignore
			Metadata: v1.RoleMetadata{},
			Spec: v1.RoleSpec{
				DisplayName: "Test Viewer Role",
				Description: "A test role for viewing agents",
				Permissions: []string{},
				IsBuiltIn:   false,
			},
			//exhaustruct:ignore
			Status: v1.RoleStatus{},
		})
		require.NoError(t, err)
		assert.NotEmpty(t, role.Metadata.UID)
		assert.Equal(t, "Test Viewer Role", role.Spec.DisplayName)
		createdRoleUID = role.Metadata.UID
	})

	// Test 4: List roles
	t.Run("ListRoles", func(t *testing.T) {
		resp, err := opampClient.RoleService.ListRoles(t.Context())
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(resp.Items), 1, "At least the created role should exist")
	})

	// Test 5: Get role by ID
	t.Run("GetRoleByID", func(t *testing.T) {
		if createdRoleUID == "" {
			t.Skip("No role created in previous test")
		}

		role, err := opampClient.RoleService.GetRole(t.Context(), createdRoleUID)
		require.NoError(t, err)
		assert.Equal(t, createdRoleUID, role.Metadata.UID)
		assert.Equal(t, "Test Viewer Role", role.Spec.DisplayName)
	})

	// Test 6: Create RoleBinding to bind role to user
	t.Run("CreateRoleBinding", func(t *testing.T) {
		if createdRoleUID == "" {
			t.Skip("No role created in previous test")
		}

		_, err := opampClient.RoleBindingService.CreateRoleBinding(t.Context(), "default", &v1.RoleBinding{
			Kind:       v1.RoleBindingKind,
			APIVersion: "v1",
			Metadata: v1.RoleBindingMetadata{
				Namespace: "default",
				Name:      "test-viewer-binding",
				//exhaustruct:ignore
				CreatedAt: v1.Time{},
				//exhaustruct:ignore
				UpdatedAt: v1.Time{},
			},
			Spec: v1.RoleBindingSpec{
				RoleRef:  v1.RoleBindingRoleRef{Kind: "Role", Name: "Test Viewer Role"},
				Subjects: []v1.RoleBindingSubject{{Kind: "User", Name: "test@test.com"}},
			},
			//exhaustruct:ignore
			Status: v1.RoleBindingStatus{},
		})
		require.NoError(t, err)
	})

	// Test 7: Delete RoleBinding
	t.Run("DeleteRoleBinding", func(t *testing.T) {
		if createdRoleUID == "" {
			t.Skip("No role created in previous test")
		}

		err := opampClient.RoleBindingService.DeleteRoleBinding(t.Context(), "default", "test-viewer-binding")
		require.NoError(t, err)
	})

	// Test 8: Delete role
	t.Run("DeleteRole", func(t *testing.T) {
		if createdRoleUID == "" {
			t.Skip("No role created in previous test")
		}

		err := opampClient.RoleService.DeleteRole(t.Context(), createdRoleUID)
		require.NoError(t, err)
	})
}
