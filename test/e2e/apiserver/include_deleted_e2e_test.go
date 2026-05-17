//go:build e2e

package apiserver_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

// startIncludeDeletedAPIServer is a shared helper that brings up an isolated MongoDB
// and apiserver for the include-deleted e2e tests in this file.
func startIncludeDeletedAPIServer(t *testing.T, dbName string) *client.Client {
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

// TestE2E_AgentPackage_IncludeDeleted verifies the includeDeleted flag for agent packages.
func TestE2E_AgentPackage_IncludeDeleted(t *testing.T) {
	t.Parallel()

	c := startIncludeDeletedAPIServer(t, "opampcommander_e2e_include_deleted_agentpackage")

	name := "deleted-pkg"

	//exhaustruct:ignore
	_, err := c.AgentPackageService.CreateAgentPackage(t.Context(), "default", &v1.AgentPackage{
		Metadata: v1.AgentPackageMetadata{Name: name, Namespace: "default"},
		Spec:     v1.AgentPackageSpec{PackageType: "TopLevelPackageName", Version: "1.0.0"},
	})
	require.NoError(t, err)

	err = c.AgentPackageService.DeleteAgentPackage(t.Context(), "default", name)
	require.NoError(t, err)

	assertAbsent(t,
		func() ([]string, error) {
			resp, err := c.AgentPackageService.ListAgentPackages(t.Context(), "default")
			if err != nil {
				return nil, err
			}

			names := make([]string, len(resp.Items))
			for i, item := range resp.Items {
				names[i] = item.Metadata.Name
			}

			return names, nil
		},
		name, "deleted agent package without includeDeleted")

	assertPresent(t,
		func() ([]string, error) {
			resp, err := c.AgentPackageService.ListAgentPackages(t.Context(), "default", client.WithIncludeDeleted(true))
			if err != nil {
				return nil, err
			}

			names := make([]string, len(resp.Items))
			for i, item := range resp.Items {
				names[i] = item.Metadata.Name
			}

			return names, nil
		},
		name, "deleted agent package with includeDeleted=true")

	_, err = c.AgentPackageService.GetAgentPackage(t.Context(), "default", name)
	require.Error(t, err, "Get deleted agent package without includeDeleted should fail")

	got, err := c.AgentPackageService.GetAgentPackage(
		t.Context(), "default", name, client.WithGetIncludeDeleted(true),
	)
	require.NoError(t, err, "Get deleted agent package with includeDeleted=true should succeed")
	assert.Equal(t, name, got.Metadata.Name)
	assert.NotNil(t, got.Metadata.DeletedAt)
}

// TestE2E_AgentRemoteConfig_IncludeDeleted verifies the includeDeleted flag for agent remote configs.
func TestE2E_AgentRemoteConfig_IncludeDeleted(t *testing.T) {
	t.Parallel()

	c := startIncludeDeletedAPIServer(t, "opampcommander_e2e_include_deleted_agentremoteconfig")

	name := "deleted-config"

	//exhaustruct:ignore
	_, err := c.AgentRemoteConfigService.CreateAgentRemoteConfig(t.Context(), "default", &v1.AgentRemoteConfig{
		Metadata: v1.AgentRemoteConfigMetadata{Name: name, Namespace: "default"},
		Spec:     v1.AgentRemoteConfigSpec{Value: "key: value", ContentType: "application/yaml"},
	})
	require.NoError(t, err)

	err = c.AgentRemoteConfigService.DeleteAgentRemoteConfig(t.Context(), "default", name)
	require.NoError(t, err)

	assertAbsent(t,
		func() ([]string, error) {
			resp, err := c.AgentRemoteConfigService.ListAgentRemoteConfigs(t.Context(), "default")
			if err != nil {
				return nil, err
			}

			names := make([]string, len(resp.Items))
			for i, item := range resp.Items {
				names[i] = item.Metadata.Name
			}

			return names, nil
		},
		name, "deleted agent remote config without includeDeleted")

	assertPresent(t,
		func() ([]string, error) {
			resp, err := c.AgentRemoteConfigService.ListAgentRemoteConfigs(
				t.Context(), "default", client.WithIncludeDeleted(true),
			)
			if err != nil {
				return nil, err
			}

			names := make([]string, len(resp.Items))
			for i, item := range resp.Items {
				names[i] = item.Metadata.Name
			}

			return names, nil
		},
		name, "deleted agent remote config with includeDeleted=true")

	_, err = c.AgentRemoteConfigService.GetAgentRemoteConfig(t.Context(), "default", name)
	require.Error(t, err)

	got, err := c.AgentRemoteConfigService.GetAgentRemoteConfig(
		t.Context(), "default", name, client.WithGetIncludeDeleted(true),
	)
	require.NoError(t, err)
	assert.Equal(t, name, got.Metadata.Name)
}

// TestE2E_Certificate_IncludeDeleted verifies the includeDeleted flag for certificates.
func TestE2E_Certificate_IncludeDeleted(t *testing.T) {
	t.Parallel()

	c := startIncludeDeletedAPIServer(t, "opampcommander_e2e_include_deleted_certificate")

	certs := generateTestCertificates(t)
	name := "deleted-cert"

	//exhaustruct:ignore
	_, err := c.CertificateService.CreateCertificate(t.Context(), "default", &v1.Certificate{
		Metadata: v1.CertificateMetadata{Name: name, Namespace: "default"},
		Spec: v1.CertificateSpec{
			Cert:       certs.CertPEM,
			PrivateKey: certs.KeyPEM,
			CaCert:     certs.CaCertPEM,
		},
	})
	require.NoError(t, err)

	err = c.CertificateService.DeleteCertificate(t.Context(), "default", name)
	require.NoError(t, err)

	assertAbsent(t,
		func() ([]string, error) {
			resp, err := c.CertificateService.ListCertificates(t.Context(), "default")
			if err != nil {
				return nil, err
			}

			names := make([]string, len(resp.Items))
			for i, item := range resp.Items {
				names[i] = item.Metadata.Name
			}

			return names, nil
		},
		name, "deleted certificate without includeDeleted")

	assertPresent(t,
		func() ([]string, error) {
			resp, err := c.CertificateService.ListCertificates(t.Context(), "default", client.WithIncludeDeleted(true))
			if err != nil {
				return nil, err
			}

			names := make([]string, len(resp.Items))
			for i, item := range resp.Items {
				names[i] = item.Metadata.Name
			}

			return names, nil
		},
		name, "deleted certificate with includeDeleted=true")

	_, err = c.CertificateService.GetCertificate(t.Context(), "default", name)
	require.Error(t, err)

	got, err := c.CertificateService.GetCertificate(t.Context(), "default", name, client.WithGetIncludeDeleted(true))
	require.NoError(t, err)
	assert.Equal(t, name, got.Metadata.Name)
	assert.NotNil(t, got.Metadata.DeletedAt)
}

// TestE2E_Namespace_IncludeDeleted verifies the includeDeleted flag for namespaces.
func TestE2E_Namespace_IncludeDeleted(t *testing.T) {
	t.Parallel()

	c := startIncludeDeletedAPIServer(t, "opampcommander_e2e_include_deleted_namespace")

	name := "deleted-namespace"

	//exhaustruct:ignore
	_, err := c.NamespaceService.CreateNamespace(t.Context(), &v1.Namespace{
		Metadata: v1.NamespaceMetadata{Name: name},
	})
	require.NoError(t, err)

	err = c.NamespaceService.DeleteNamespace(t.Context(), name)
	require.NoError(t, err)

	assertAbsent(t,
		func() ([]string, error) {
			resp, err := c.NamespaceService.ListNamespaces(t.Context())
			if err != nil {
				return nil, err
			}

			names := make([]string, len(resp.Items))
			for i, item := range resp.Items {
				names[i] = item.Metadata.Name
			}

			return names, nil
		},
		name, "deleted namespace without includeDeleted")

	assertPresent(t,
		func() ([]string, error) {
			resp, err := c.NamespaceService.ListNamespaces(t.Context(), client.WithIncludeDeleted(true))
			if err != nil {
				return nil, err
			}

			names := make([]string, len(resp.Items))
			for i, item := range resp.Items {
				names[i] = item.Metadata.Name
			}

			return names, nil
		},
		name, "deleted namespace with includeDeleted=true")

	_, err = c.NamespaceService.GetNamespace(t.Context(), name)
	require.Error(t, err)

	got, err := c.NamespaceService.GetNamespace(t.Context(), name, client.WithGetIncludeDeleted(true))
	require.NoError(t, err)
	assert.Equal(t, name, got.Metadata.Name)
	assert.NotNil(t, got.Metadata.DeletedAt)
}

// TestE2E_Role_IncludeDeleted verifies the includeDeleted flag for roles.
func TestE2E_Role_IncludeDeleted(t *testing.T) {
	t.Parallel()

	c := startIncludeDeletedAPIServer(t, "opampcommander_e2e_include_deleted_role")

	//exhaustruct:ignore
	created, err := c.RoleService.CreateRole(t.Context(), &v1.Role{
		Spec: v1.RoleSpec{
			DisplayName: "deleted-role-display",
			Description: "role to be deleted",
			Permissions: []string{"agent:read"},
		},
	})
	require.NoError(t, err)

	uid := created.Metadata.UID

	err = c.RoleService.DeleteRole(t.Context(), uid)
	require.NoError(t, err)

	assertAbsent(t,
		func() ([]string, error) {
			resp, err := c.RoleService.ListRoles(t.Context())
			if err != nil {
				return nil, err
			}

			uids := make([]string, len(resp.Items))
			for i, item := range resp.Items {
				uids[i] = item.Metadata.UID
			}

			return uids, nil
		},
		uid, "deleted role without includeDeleted")

	assertPresent(t,
		func() ([]string, error) {
			resp, err := c.RoleService.ListRoles(t.Context(), client.WithIncludeDeleted(true))
			if err != nil {
				return nil, err
			}

			uids := make([]string, len(resp.Items))
			for i, item := range resp.Items {
				uids[i] = item.Metadata.UID
			}

			return uids, nil
		},
		uid, "deleted role with includeDeleted=true")

	_, err = c.RoleService.GetRole(t.Context(), uid)
	require.Error(t, err)

	got, err := c.RoleService.GetRole(t.Context(), uid, client.WithGetIncludeDeleted(true))
	require.NoError(t, err)
	assert.Equal(t, uid, got.Metadata.UID)
	assert.NotNil(t, got.Metadata.DeletedAt)
}

// TestE2E_RoleBinding_IncludeDeleted verifies the includeDeleted flag for role bindings.
func TestE2E_RoleBinding_IncludeDeleted(t *testing.T) {
	t.Parallel()

	c := startIncludeDeletedAPIServer(t, "opampcommander_e2e_include_deleted_rolebinding")

	// RoleBinding creation validates that the referenced Role exists, so create one first.
	roleName := "Include Deleted Test Role"

	//exhaustruct:ignore
	_, err := c.RoleService.CreateRole(t.Context(), &v1.Role{
		Spec: v1.RoleSpec{
			DisplayName: roleName,
			Description: "role for include-deleted rolebinding test",
			Permissions: []string{"agent:read"},
		},
	})
	require.NoError(t, err)

	name := "deleted-binding"

	//exhaustruct:ignore
	_, err = c.RoleBindingService.CreateRoleBinding(t.Context(), "default", &v1.RoleBinding{
		Metadata: v1.RoleBindingMetadata{Name: name, Namespace: "default"},
		Spec: v1.RoleBindingSpec{
			RoleRef: v1.RoleBindingRoleRef{Kind: "Role", Name: roleName},
			Subjects: []v1.RoleBindingSubject{
				{Kind: "User", Name: "user@example.com"},
			},
		},
	})
	require.NoError(t, err)

	err = c.RoleBindingService.DeleteRoleBinding(t.Context(), "default", name)
	require.NoError(t, err)

	assertAbsent(t,
		func() ([]string, error) {
			resp, err := c.RoleBindingService.ListRoleBindings(t.Context(), "default")
			if err != nil {
				return nil, err
			}

			names := make([]string, len(resp.Items))
			for i, item := range resp.Items {
				names[i] = item.Metadata.Name
			}

			return names, nil
		},
		name, "deleted role binding without includeDeleted")

	assertPresent(t,
		func() ([]string, error) {
			resp, err := c.RoleBindingService.ListRoleBindings(t.Context(), "default", client.WithIncludeDeleted(true))
			if err != nil {
				return nil, err
			}

			names := make([]string, len(resp.Items))
			for i, item := range resp.Items {
				names[i] = item.Metadata.Name
			}

			return names, nil
		},
		name, "deleted role binding with includeDeleted=true")

	_, err = c.RoleBindingService.GetRoleBinding(t.Context(), "default", name)
	require.Error(t, err)

	got, err := c.RoleBindingService.GetRoleBinding(
		t.Context(), "default", name, client.WithGetIncludeDeleted(true),
	)
	require.NoError(t, err)
	assert.Equal(t, name, got.Metadata.Name)
	assert.NotNil(t, got.Metadata.DeletedAt)
}

// TestE2E_User_IncludeDeleted verifies the includeDeleted flag for users.
func TestE2E_User_IncludeDeleted(t *testing.T) {
	t.Parallel()

	c := startIncludeDeletedAPIServer(t, "opampcommander_e2e_include_deleted_user")

	//exhaustruct:ignore
	created, err := c.UserService.CreateUser(t.Context(), &v1.User{
		Spec: v1.UserSpec{
			Email:    "deleted-user@example.com",
			Username: "deleted-user",
			IsActive: true,
		},
	})
	require.NoError(t, err)

	uid := created.Metadata.UID
	require.NotEmpty(t, uid)

	// Validate uid is a uuid before deleting so we fail fast on malformed responses.
	_, err = uuid.Parse(uid)
	require.NoError(t, err)

	err = c.UserService.DeleteUser(t.Context(), uid)
	require.NoError(t, err)

	assertAbsent(t,
		func() ([]string, error) {
			resp, err := c.UserService.ListUsers(t.Context())
			if err != nil {
				return nil, err
			}

			uids := make([]string, len(resp.Items))
			for i, item := range resp.Items {
				uids[i] = item.Metadata.UID
			}

			return uids, nil
		},
		uid, "deleted user without includeDeleted")

	assertPresent(t,
		func() ([]string, error) {
			resp, err := c.UserService.ListUsers(t.Context(), client.WithIncludeDeleted(true))
			if err != nil {
				return nil, err
			}

			uids := make([]string, len(resp.Items))
			for i, item := range resp.Items {
				uids[i] = item.Metadata.UID
			}

			return uids, nil
		},
		uid, "deleted user with includeDeleted=true")

	_, err = c.UserService.GetUser(t.Context(), uid)
	require.Error(t, err)

	got, err := c.UserService.GetUser(t.Context(), uid, client.WithGetIncludeDeleted(true))
	require.NoError(t, err)
	assert.Equal(t, uid, got.Metadata.UID)
	assert.NotNil(t, got.Metadata.DeletedAt)
}

func assertAbsent(t *testing.T, fetch func() ([]string, error), needle, ctxLabel string) {
	t.Helper()

	items, err := fetch()
	require.NoError(t, err, "fetch for %q", ctxLabel)

	for _, item := range items {
		if item == needle {
			t.Fatalf("%s: expected %q to be absent but it was present", ctxLabel, needle)
		}
	}
}

func assertPresent(t *testing.T, fetch func() ([]string, error), needle, ctxLabel string) {
	t.Helper()

	items, err := fetch()
	require.NoError(t, err, "fetch for %q", ctxLabel)

	for _, item := range items {
		if item == needle {
			return
		}
	}

	t.Fatalf("%s: expected %q to be present but it was absent (have %v)", ctxLabel, needle, items)
}
