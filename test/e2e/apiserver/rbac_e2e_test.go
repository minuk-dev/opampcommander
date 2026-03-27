//go:build e2e

package apiserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

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

	token := getAuthToken(t, apiBaseURL)

	// Test 1: Get current user profile via /api/v1/users/me
	t.Run("GetCurrentUserProfile", func(t *testing.T) {
		profile := getUserProfile(t, apiBaseURL, token)
		assert.NotEmpty(t, profile.User.Metadata.UID)
		assert.Equal(t, "test@test.com", profile.User.Spec.Email)
		assert.Equal(t, "test-admin", profile.User.Spec.Username)
		assert.True(t, profile.User.Spec.IsActive)
	})

	// Test 2: List users
	t.Run("ListUsers", func(t *testing.T) {
		users := listUsers(t, apiBaseURL, token)
		assert.GreaterOrEqual(t, len(users), 1, "At least the admin user should exist")
	})

	// Test 3: Create a role
	var createdRoleUID string

	t.Run("CreateRole", func(t *testing.T) {
		role := createRole(t, apiBaseURL, token, &v1.Role{
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
		roles := listRoles(t, apiBaseURL, token)
		assert.GreaterOrEqual(t, len(roles), 1, "At least the created role should exist")
	})

	// Test 5: Get role by ID
	t.Run("GetRoleByID", func(t *testing.T) {
		if createdRoleUID == "" {
			t.Skip("No role created in previous test")
		}

		role := getRoleByID(t, apiBaseURL, token, createdRoleUID)
		assert.Equal(t, createdRoleUID, role.Metadata.UID)
		assert.Equal(t, "Test Viewer Role", role.Spec.DisplayName)
	})

	// Test 6: Assign role to user
	t.Run("AssignRoleToUser", func(t *testing.T) {
		if createdRoleUID == "" {
			t.Skip("No role created in previous test")
		}

		profile := getUserProfile(t, apiBaseURL, token)
		userUID := profile.User.Metadata.UID

		assignRole(t, apiBaseURL, token, userUID, createdRoleUID)
	})

	// Test 7: Get user roles via RBAC endpoint
	t.Run("GetUserRoles", func(t *testing.T) {
		if createdRoleUID == "" {
			t.Skip("No role created in previous test")
		}

		profile := getUserProfile(t, apiBaseURL, token)
		userUID := profile.User.Metadata.UID

		roles := getUserRoles(t, apiBaseURL, token, userUID)
		assert.GreaterOrEqual(t, len(roles), 1, "User should have at least one role assigned")
	})

	// Test 8: Check permission
	t.Run("CheckPermission", func(t *testing.T) {
		profile := getUserProfile(t, apiBaseURL, token)
		userUID := profile.User.Metadata.UID

		result := checkPermission(t, apiBaseURL, token, userUID, "agent", "read")
		// Result may be true or false depending on policy setup; just verify it returns without error
		assert.IsType(t, true, result.Allowed)
	})

	// Test 9: Unassign role from user
	t.Run("UnassignRoleFromUser", func(t *testing.T) {
		if createdRoleUID == "" {
			t.Skip("No role created in previous test")
		}

		profile := getUserProfile(t, apiBaseURL, token)
		userUID := profile.User.Metadata.UID

		unassignRole(t, apiBaseURL, token, userUID, createdRoleUID)
	})

	// Test 10: Delete role
	t.Run("DeleteRole", func(t *testing.T) {
		if createdRoleUID == "" {
			t.Skip("No role created in previous test")
		}

		deleteRole(t, apiBaseURL, token, createdRoleUID)
	})
}

func getUserProfile(t *testing.T, baseURL, token string) v1.UserProfileResponse {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/v1/users/me", nil) //nolint:noctx
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result v1.UserProfileResponse
	require.NoError(t, json.Unmarshal(body, &result))

	return result
}

func listUsers(t *testing.T, baseURL, token string) []v1.User {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/v1/users", nil) //nolint:noctx
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result struct {
		Items []v1.User `json:"items"`
	}
	require.NoError(t, json.Unmarshal(body, &result))

	return result.Items
}

func createRole(t *testing.T, baseURL, token string, role *v1.Role) v1.Role {
	t.Helper()

	roleJSON, err := json.Marshal(role)
	require.NoError(t, err)

	req, err := http.NewRequest( //nolint:noctx
		http.MethodPost, baseURL+"/api/v1/roles", bytes.NewReader(roleJSON),
	)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result v1.Role
	require.NoError(t, json.Unmarshal(body, &result))

	return result
}

func listRoles(t *testing.T, baseURL, token string) []v1.Role {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/v1/roles", nil) //nolint:noctx
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result struct {
		Items []v1.Role `json:"items"`
	}
	require.NoError(t, json.Unmarshal(body, &result))

	return result.Items
}

func getRoleByID(t *testing.T, baseURL, token, roleUID string) v1.Role {
	t.Helper()

	url := fmt.Sprintf("%s/api/v1/roles/%s", baseURL, roleUID)

	req, err := http.NewRequest(http.MethodGet, url, nil) //nolint:noctx
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result v1.Role
	require.NoError(t, json.Unmarshal(body, &result))

	return result
}

func assignRole(t *testing.T, baseURL, token, userID, roleID string) {
	t.Helper()

	reqBody := v1.AssignRoleRequest{
		UserID: userID,
		RoleID: roleID,
	}

	bodyJSON, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req, err := http.NewRequest( //nolint:noctx
		http.MethodPost, baseURL+"/api/v1/rbac/assign", bytes.NewReader(bodyJSON),
	)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func getUserRoles(t *testing.T, baseURL, token, userID string) []v1.Role {
	t.Helper()

	url := fmt.Sprintf("%s/api/v1/rbac/users/%s/roles", baseURL, userID)

	req, err := http.NewRequest(http.MethodGet, url, nil) //nolint:noctx
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result struct {
		Items []v1.Role `json:"items"`
	}
	require.NoError(t, json.Unmarshal(body, &result))

	return result.Items
}

func checkPermission(
	t *testing.T,
	baseURL, token, userID, resource, action string,
) v1.CheckPermissionResponse {
	t.Helper()

	reqBody := v1.CheckPermissionRequest{
		UserID:   userID,
		Resource: resource,
		Action:   action,
	}

	bodyJSON, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req, err := http.NewRequest( //nolint:noctx
		http.MethodPost, baseURL+"/api/v1/rbac/check", bytes.NewReader(bodyJSON),
	)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result v1.CheckPermissionResponse
	require.NoError(t, json.Unmarshal(body, &result))

	return result
}

func unassignRole(t *testing.T, baseURL, token, userID, roleID string) {
	t.Helper()

	reqBody := struct {
		UserID string `json:"userId"`
		RoleID string `json:"roleId"`
	}{
		UserID: userID,
		RoleID: roleID,
	}

	bodyJSON, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req, err := http.NewRequest( //nolint:noctx
		http.MethodPost, baseURL+"/api/v1/rbac/unassign", bytes.NewReader(bodyJSON),
	)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func deleteRole(t *testing.T, baseURL, token, roleID string) {
	t.Helper()

	url := fmt.Sprintf("%s/api/v1/roles/%s", baseURL, roleID)

	req, err := http.NewRequest(http.MethodDelete, url, nil) //nolint:noctx
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusNoContent, resp.StatusCode)
}
