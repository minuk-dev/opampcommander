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

	deleteTargetAgentPackage, readErr := c.AgentPackageService.GetAgentPackage(t.Context(), "default", name)
	require.NoError(t, readErr)
	err = c.AgentPackageService.DeleteAgentPackage(t.Context(), "default", name, deleteTargetAgentPackage.Metadata.ResourceVersion)
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

	deleteTargetAgentRemoteConfig, readErr := c.AgentRemoteConfigService.GetAgentRemoteConfig(t.Context(), "default", name)
	require.NoError(t, readErr)
	err = c.AgentRemoteConfigService.DeleteAgentRemoteConfig(t.Context(), "default", name, deleteTargetAgentRemoteConfig.Metadata.ResourceVersion)
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

	deleteTargetNamespace, readErr := c.NamespaceService.GetNamespace(t.Context(), name)
	require.NoError(t, readErr)
	err = c.NamespaceService.DeleteNamespace(t.Context(), name, deleteTargetNamespace.Metadata.ResourceVersion)
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
