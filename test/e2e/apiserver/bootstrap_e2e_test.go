//go:build e2e

package apiserver_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"

	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

// TestE2E_APIServer_BootstrapReconcile verifies namespace bootstrap across restarts.
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

	defaultNS.Metadata.Labels = map[string]string{"test-edit": "value"}
	_, err = client1.NamespaceService.UpdateNamespace(t.Context(), "default", defaultNS)
	require.NoError(t, err)
	srv1.Stop()

	srv2 := base.StartAPIServer(mongo.URI, dbName)
	srv2.WaitForReady()
	reconciled, err := srv2.Client().NamespaceService.GetNamespace(t.Context(), "default")
	require.NoError(t, err)
	assert.NotContains(t, reconciled.Metadata.Labels, "test-edit")
}
