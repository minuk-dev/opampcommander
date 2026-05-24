//go:build e2e

package apiserver_test

import (
	"errors"
	"net/http"
	"slices"
	"testing"

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

	// DefaultRole_HasBuiltInReadPermissions verifies that the built-in "default" role exists
	// at startup with GET/LIST permissions on the resources every authenticated user needs
	// to use the dashboard / CLI. Together with the SyncPolicies hook that auto-grants the
	// default role to every user in the "default" namespace, this is what gives every
	// authenticated user baseline read access there.
	t.Run("DefaultRole_HasBuiltInReadPermissions", func(t *testing.T) {
		expected := []string{
			"agent:GET", "agent:LIST",
			"agentgroup:GET", "agentgroup:LIST",
			"agentpackage:GET", "agentpackage:LIST",
			"agentremoteconfig:GET", "agentremoteconfig:LIST",
			"connection:GET", "connection:LIST",
			"rolebinding:GET",
		}

		// Permissions that must NOT appear on the default role. Global resources
		// (server, role) are intentionally excluded because the default role is
		// grouped to the "default" namespace only — granting global perms here
		// would be unreachable in Casbin and is reserved for admins.
		forbidden := []string{
			"certificate:GET", "certificate:LIST",
			"server:GET", "server:LIST",
			"role:GET", "role:LIST",
			"role:CREATE", "role:UPDATE", "role:DELETE",
			"rolebinding:CREATE", "rolebinding:UPDATE", "rolebinding:DELETE",
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

		for _, name := range forbidden {
			assert.False(t, slices.Contains(defaultRole.Spec.Permissions, name),
				"default role must not grant %q (got %v)",
				name, defaultRole.Spec.Permissions)
		}
	})

	// DefaultRole_RuntimeEnforcement asserts that the permission strings on the default
	// role actually translate into allowed/denied responses at the HTTP layer for a
	// non-admin user. It guards against regressions where the role lists the right
	// permissions but Casbin (matcher, grouping domain) silently blocks them — which
	// the prior assertion of role.Spec.Permissions would not catch.
	t.Run("DefaultRole_RuntimeEnforcement", func(t *testing.T) {
		const nonAdminEmail = "regular@user.example.com"

		created, err := opampClient.UserService.CreateUser(t.Context(), &v1.User{
			Kind:       v1.UserKind,
			APIVersion: "v1",
			//exhaustruct:ignore
			Metadata: v1.UserMetadata{},
			Spec: v1.UserSpec{
				Email:    nonAdminEmail,
				Username: "regular-user",
				IsActive: true,
			},
			//exhaustruct:ignore
			Status: v1.UserStatus{},
		})
		require.NoError(t, err, "admin must be able to create the non-admin test user")
		require.NotEmpty(t, created.Metadata.UID)

		nonAdminClient := apiServer.ClientAs(nonAdminEmail)

		t.Run("Allowed: list agents in default namespace", func(t *testing.T) {
			_, err := nonAdminClient.AgentService.ListAgents(t.Context(), "default")
			assert.NoError(t, err, "default role grants agent:LIST in default namespace")
		})

		t.Run("Allowed: list agentgroups in default namespace", func(t *testing.T) {
			_, err := nonAdminClient.AgentGroupService.ListAgentGroups(t.Context(), "default")
			assert.NoError(t, err, "default role grants agentgroup:LIST in default namespace")
		})

		t.Run("Allowed: list connections in default namespace", func(t *testing.T) {
			_, err := nonAdminClient.ConnectionService.ListConnections(t.Context(), "default")
			assert.NoError(t, err, "default role grants connection:LIST in default namespace")
		})

		t.Run("Forbidden: list certificates (removed from default floor)", func(t *testing.T) {
			_, err := nonAdminClient.CertificateService.ListCertificates(t.Context(), "default")
			assertHTTPForbidden(t, err)
		})

		t.Run("Forbidden: list servers (global, admin-only)", func(t *testing.T) {
			// /api/v1/servers has no dedicated client method — issue a raw GET with
			// the non-admin token so we exercise the same RBAC middleware path.
			status := nonAdminGetStatus(t, apiServer, nonAdminEmail, "/api/v1/servers")
			assert.Equal(t, http.StatusForbidden, status, "default users must not be able to list servers")
		})

		t.Run("Forbidden: list roles (global, admin-only)", func(t *testing.T) {
			_, err := nonAdminClient.RoleService.ListRoles(t.Context())
			assertHTTPForbidden(t, err)
		})

		t.Run("Forbidden: list agents in non-default namespace", func(t *testing.T) {
			_, err := nonAdminClient.AgentService.ListAgents(t.Context(), "other-namespace")
			assertHTTPForbidden(t, err)
		})

		t.Run("Forbidden: create agentgroup in default namespace (no write perm)", func(t *testing.T) {
			//exhaustruct:ignore
			_, err := nonAdminClient.AgentGroupService.CreateAgentGroup(t.Context(), "default", &v1.AgentGroup{
				Kind:       v1.AgentGroupKind,
				APIVersion: "v1",
				Metadata: v1.Metadata{
					Namespace: "default",
					Name:      "test-blocked",
				},
			})
			assertHTTPForbidden(t, err)
		})
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

// assertHTTPForbidden fails the test unless err is a client.ResponseError carrying
// a 403 status code. Surfaces a clear message when the response is wrong (e.g. a
// 500 from the server slipping through as "blocked").
func assertHTTPForbidden(t *testing.T, err error) {
	t.Helper()

	require.Error(t, err, "expected a 403 response, got success")

	var respErr *client.ResponseError
	require.True(t, errors.As(err, &respErr), "expected client.ResponseError, got %T: %v", err, err)
	assert.Equal(t, http.StatusForbidden, respErr.StatusCode, "expected 403; full error: %v", err)
}

// nonAdminGetStatus issues a GET to the given path as the given email and returns
// the response status code. Used for endpoints with no dedicated client method.
func nonAdminGetStatus(t *testing.T, apiServer *testutil.APIServer, email, path string) int {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, apiServer.Endpoint+path, nil)
	require.NoError(t, err)

	req.Header.Set("Authorization", "Bearer "+apiServer.IssueTokenForEmail(email))

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()

	return res.StatusCode
}
