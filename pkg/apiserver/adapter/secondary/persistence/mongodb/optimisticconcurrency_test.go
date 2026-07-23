package mongodb_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	mongoTestContainer "github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/mongodb"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

// TestOptimisticConcurrency verifies that the agentpackage, host, and endpoint
// adapters reject a stale write with model.ErrConflict instead of silently
// clobbering a concurrent writer, and bump the caller's version on success —
// consistent with the agent adapter's optimistic-concurrency test.
func TestOptimisticConcurrency(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	t.Parallel()
	base := testutil.NewBase(t)
	ctx := t.Context()
	mongoDBContainer, err := mongoTestContainer.Run(ctx, testMongoDBImage)
	require.NoError(t, err)

	mongoDBURI, err := mongoDBContainer.ConnectionString(ctx)
	require.NoError(t, err)

	client, err := mongo.Connect(options.Client().ApplyURI(mongoDBURI))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, client.Disconnect(ctx))
	})

	database := client.Database("testdb")

	t.Run("agent package", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		repo := mongodb.NewAgentPackageRepository(database, base.Logger)

		pkg := &agentmodel.AgentPackage{
			Metadata: agentmodel.AgentPackageMetadata{Name: "otelcol", Namespace: "default"},
			Spec:     agentmodel.AgentPackageSpec{PackageType: "top", Version: "1.0.0"},
			Status:   agentmodel.AgentPackageStatus{},
		}
		require.Equal(t, int64(0), pkg.Metadata.ResourceVersion)

		saved, err := repo.PutAgentPackage(ctx, pkg)
		require.NoError(t, err)
		assert.Equal(t, int64(1), saved.Metadata.ResourceVersion)

		loadA, err := repo.GetAgentPackage(ctx, "default", "otelcol", nil)
		require.NoError(t, err)
		loadB, err := repo.GetAgentPackage(ctx, "default", "otelcol", nil)
		require.NoError(t, err)

		loadA.Spec.Version = "2.0.0"
		_, err = repo.PutAgentPackage(ctx, loadA)
		require.NoError(t, err)

		loadB.Spec.Version = "3.0.0"
		_, err = repo.PutAgentPackage(ctx, loadB)
		require.ErrorIs(t, err, model.ErrConflict)

		stored, err := repo.GetAgentPackage(ctx, "default", "otelcol", nil)
		require.NoError(t, err)
		assert.Equal(t, "2.0.0", stored.Spec.Version)
		assert.Equal(t, int64(2), stored.Metadata.ResourceVersion)
	})

	t.Run("host", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		repo := mongodb.NewHostRepository(database, base.Logger)
		now := time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC)

		host := agentmodel.NewHost("host-1", now)
		require.Equal(t, int64(0), host.Metadata.ResourceVersion)

		saved, err := repo.PutHost(ctx, host)
		require.NoError(t, err)
		assert.Equal(t, int64(1), saved.Metadata.ResourceVersion)

		loadA, err := repo.GetHost(ctx, "host-1")
		require.NoError(t, err)
		loadB, err := repo.GetHost(ctx, "host-1")
		require.NoError(t, err)

		loadA.Metadata.Name = "host-a"
		_, err = repo.PutHost(ctx, loadA)
		require.NoError(t, err)

		loadB.Metadata.Name = "host-b"
		_, err = repo.PutHost(ctx, loadB)
		require.ErrorIs(t, err, model.ErrConflict)

		stored, err := repo.GetHost(ctx, "host-1")
		require.NoError(t, err)
		assert.Equal(t, "host-a", stored.Metadata.Name)
		assert.Equal(t, int64(2), stored.Metadata.ResourceVersion)
	})

	t.Run("container", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		repo := mongodb.NewContainerRepository(database, base.Logger)
		now := time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC)

		container := agentmodel.NewContainer("container-1", now)
		require.Equal(t, int64(0), container.Metadata.ResourceVersion)

		saved, err := repo.PutContainer(ctx, container)
		require.NoError(t, err)
		assert.Equal(t, int64(1), saved.Metadata.ResourceVersion)

		loadA, err := repo.GetContainer(ctx, "container-1")
		require.NoError(t, err)
		loadB, err := repo.GetContainer(ctx, "container-1")
		require.NoError(t, err)

		loadA.Metadata.Name = "cont-a"
		_, err = repo.PutContainer(ctx, loadA)
		require.NoError(t, err)

		loadB.Metadata.Name = "cont-b"
		_, err = repo.PutContainer(ctx, loadB)
		require.ErrorIs(t, err, model.ErrConflict)

		stored, err := repo.GetContainer(ctx, "container-1")
		require.NoError(t, err)
		assert.Equal(t, "cont-a", stored.Metadata.Name)
		assert.Equal(t, int64(2), stored.Metadata.ResourceVersion)
	})

	t.Run("endpoint", func(t *testing.T) {
		t.Parallel()
		ctx := t.Context()
		repo := mongodb.NewEndpointRepository(database, base.Logger)
		now := time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC)

		endpoint := agentmodel.NewEndpoint("default", "tempo", nil, now, "tester")
		require.Equal(t, int64(0), endpoint.Metadata.ResourceVersion)

		saved, err := repo.PutEndpoint(ctx, endpoint)
		require.NoError(t, err)
		assert.Equal(t, int64(1), saved.Metadata.ResourceVersion)

		loadA, err := repo.GetEndpoint(ctx, "default", "tempo", nil)
		require.NoError(t, err)
		loadB, err := repo.GetEndpoint(ctx, "default", "tempo", nil)
		require.NoError(t, err)

		loadA.Spec.URL = "https://a.example.com"
		_, err = repo.PutEndpoint(ctx, loadA)
		require.NoError(t, err)

		loadB.Spec.URL = "https://b.example.com"
		_, err = repo.PutEndpoint(ctx, loadB)
		require.ErrorIs(t, err, model.ErrConflict)

		stored, err := repo.GetEndpoint(ctx, "default", "tempo", nil)
		require.NoError(t, err)
		assert.Equal(t, "https://a.example.com", stored.Spec.URL)
		assert.Equal(t, int64(2), stored.Metadata.ResourceVersion)
	})
}
