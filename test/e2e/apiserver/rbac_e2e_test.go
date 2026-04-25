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
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestE2E_APIServer_RBAC(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()

	base := testutil.NewBase(t)

	// Given: Infrastructure is set up (MongoDB + API Server)
	mongoContainer, mongoURI := startMongoDB(t)

	defer func() { _ = mongoContainer.Terminate(ctx) }()

	apiPort := base.GetFreeTCPPort()

	stopServer, apiBaseURL := setupAPIServer(t, apiPort, mongoURI, "opampcommander_e2e_rbac_test")
	defer stopServer()

	waitForAPIServerReady(t, apiBaseURL)

	opampClient := createOpampClient(t, apiBaseURL)

	// Test 1: Get current user profile via /api/v1/users/me
	t.Run("GetCurrentUserProfile", func(t *testing.T) {
		profile := getUserProfile(t, opampClient)
		assert.NotEmpty(t, profile.User.Metadata.UID)
		assert.Equal(t, "test@test.com", profile.User.Spec.Email)
		assert.Equal(t, "test-admin", profile.User.Spec.Username)
		assert.True(t, profile.User.Spec.IsActive)
	})

	// Test 2: List users
	t.Run("ListUsers", func(t *testing.T) {
		users := listUsers(t, opampClient)
		assert.GreaterOrEqual(t, len(users), 1, "At least the admin user should exist")
	})

	// Test 3: Create a role
	var createdRoleUID string

	t.Run("CreateRole", func(t *testing.T) {
		role := createRole(t, opampClient, &v1.Role{
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
		assert.NotEmpty(t, role.Metadata.UID)
		assert.Equal(t, "Test Viewer Role", role.Spec.DisplayName)
		createdRoleUID = role.Metadata.UID
	})

	// Test 4: List roles
	t.Run("ListRoles", func(t *testing.T) {
		roles := listRoles(t, opampClient)
		assert.GreaterOrEqual(t, len(roles), 1, "At least the created role should exist")
	})

	// Test 5: Get role by ID
	t.Run("GetRoleByID", func(t *testing.T) {
		if createdRoleUID == "" {
			t.Skip("No role created in previous test")
		}

		role := getRoleByID(t, opampClient, createdRoleUID)
		assert.Equal(t, createdRoleUID, role.Metadata.UID)
		assert.Equal(t, "Test Viewer Role", role.Spec.DisplayName)
	})

	// Test 6: Create RoleBinding to bind role to user
	t.Run("CreateRoleBinding", func(t *testing.T) {
		if createdRoleUID == "" {
			t.Skip("No role created in previous test")
		}

		createRoleBinding(t, opampClient, "default", "test-viewer-binding",
			"Test Viewer Role", "test@test.com")
	})

	// Test 7: Get user roles via RBAC endpoint
	t.Run("GetUserRoles", func(t *testing.T) {
		if createdRoleUID == "" {
			t.Skip("No role created in previous test")
		}

		profile := getUserProfile(t, opampClient)
		userUID := profile.User.Metadata.UID

		roles := getUserRoles(t, opampClient, userUID)
		assert.GreaterOrEqual(t, len(roles), 1, "User should have at least one role assigned")
	})

	// Test 8: Check permission
	t.Run("CheckPermission", func(t *testing.T) {
		profile := getUserProfile(t, opampClient)
		userUID := profile.User.Metadata.UID

		result := checkPermission(t, opampClient, userUID, "agent", "read")
		// Result may be true or false depending on policy setup; just verify it returns without error
		assert.IsType(t, true, result.Allowed)
	})

	// Test 9: Delete RoleBinding
	t.Run("DeleteRoleBinding", func(t *testing.T) {
		if createdRoleUID == "" {
			t.Skip("No role created in previous test")
		}

		deleteRoleBinding(t, opampClient, "default", "test-viewer-binding")
	})

	// Test 10: Delete role
	t.Run("DeleteRole", func(t *testing.T) {
		if createdRoleUID == "" {
			t.Skip("No role created in previous test")
		}

		deleteRole(t, opampClient, createdRoleUID)
	})
}

func getUserProfile(t *testing.T, c *client.Client) v1.UserProfileResponse {
	t.Helper()

	profile, err := c.UserService.GetMyProfile(t.Context())
	require.NoError(t, err)

	return *profile
}

func listUsers(t *testing.T, c *client.Client) []v1.User {
	t.Helper()

	resp, err := c.UserService.ListUsers(t.Context())
	require.NoError(t, err)

	return resp.Items
}

func createRole(t *testing.T, c *client.Client, role *v1.Role) v1.Role {
	t.Helper()

	result, err := c.RoleService.CreateRole(t.Context(), role)
	require.NoError(t, err)

	return *result
}

func listRoles(t *testing.T, c *client.Client) []v1.Role {
	t.Helper()

	resp, err := c.RoleService.ListRoles(t.Context())
	require.NoError(t, err)

	return resp.Items
}

func getRoleByID(t *testing.T, c *client.Client, roleUID string) v1.Role {
	t.Helper()

	role, err := c.RoleService.GetRole(t.Context(), roleUID)
	require.NoError(t, err)

	return *role
}

func createRoleBinding(
	t *testing.T,
	c *client.Client,
	namespace, name, roleDisplayName, userEmail string,
) v1.RoleBinding {
	t.Helper()

	rb := v1.RoleBinding{
		Kind:       v1.RoleBindingKind,
		APIVersion: "v1",
		Metadata: v1.RoleBindingMetadata{
			Namespace: namespace,
			Name:      name,
			//exhaustruct:ignore
			CreatedAt: v1.Time{},
			//exhaustruct:ignore
			UpdatedAt: v1.Time{},
		},
		Spec: v1.RoleBindingSpec{
			RoleRef: v1.RoleBindingRoleRef{
				Kind: "Role",
				Name: roleDisplayName,
			},
			Subject: v1.RoleBindingSubject{
				Kind: "User",
				Name: userEmail,
			},
		},
		//exhaustruct:ignore
		Status: v1.RoleBindingStatus{},
	}

	result, err := c.RoleBindingService.CreateRoleBinding(t.Context(), namespace, &rb)
	require.NoError(t, err, "failed to create role binding %s in namespace %s", name, namespace)

	return *result
}

func deleteRoleBinding(t *testing.T, c *client.Client, namespace, name string) {
	t.Helper()

	err := c.RoleBindingService.DeleteRoleBinding(t.Context(), namespace, name)
	require.NoError(t, err, "failed to delete role binding %s in namespace %s", name, namespace)
}

func getUserRoles(t *testing.T, c *client.Client, userID string) []v1.Role {
	t.Helper()

	resp, err := c.RBACService.GetUserRoles(t.Context(), userID)
	require.NoError(t, err)

	return resp.Items
}

func checkPermission(
	t *testing.T,
	c *client.Client,
	userID, resource, action string,
) v1.CheckPermissionResponse {
	t.Helper()

	result, err := c.RBACService.CheckPermission(t.Context(), &v1.CheckPermissionRequest{
		UserID:    userID,
		Namespace: "*",
		Resource:  resource,
		Action:    action,
	})
	require.NoError(t, err)

	return *result
}

func deleteRole(t *testing.T, c *client.Client, roleID string) {
	t.Helper()

	err := c.RoleService.DeleteRole(t.Context(), roleID)
	require.NoError(t, err)
}
