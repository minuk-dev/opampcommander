package testutil

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// CheckResourceWrites runs the same atomic-write contract against both storage adapters.
func CheckResourceWrites(t *testing.T, groups agentport.AgentGroupPersistencePort,
	packages agentport.AgentPackagePersistencePort, configs agentport.AgentRemoteConfigPersistencePort,
	namespaces agentport.NamespacePersistencePort,
) {
	t.Helper()
	t.Run("agentgroup", func(t *testing.T) {
		checkVersionedWrites(t,
			func() *agentmodel.AgentGroup {
				return agentmodel.NewAgentGroup("default", "group", nil, time.Now(), "test")
			},
			func(ctx context.Context, v *agentmodel.AgentGroup) (*agentmodel.AgentGroup, error) {
				return groups.PutAgentGroup(ctx, "default", "group", v)
			},
			func(ctx context.Context) (*agentmodel.AgentGroup, error) {
				return groups.GetAgentGroup(ctx, "default", "group", nil)
			},
			func(v *agentmodel.AgentGroup) int64 { return v.Metadata.ResourceVersion },
			func(v *agentmodel.AgentGroup) { v.MarkDeleted(time.Now(), "test") })
	})
	t.Run("agentpackage", func(t *testing.T) {
		checkVersionedWrites(t,
			func() *agentmodel.AgentPackage {
				//exhaustruct:ignore
				return &agentmodel.AgentPackage{
					Metadata: agentmodel.AgentPackageMetadata{Namespace: "default", Name: "package"},
				}
			},
			packages.PutAgentPackage,
			func(ctx context.Context) (*agentmodel.AgentPackage, error) {
				return packages.GetAgentPackage(ctx, "default", "package", nil)
			},
			func(v *agentmodel.AgentPackage) int64 { return v.Metadata.ResourceVersion },
			func(v *agentmodel.AgentPackage) { v.MarkAsDeleted(time.Now(), "test") })
	})
	t.Run("remoteconfig", func(t *testing.T) {
		checkVersionedWrites(t,
			func() *agentmodel.AgentRemoteConfig {
				//exhaustruct:ignore
				return &agentmodel.AgentRemoteConfig{
					Metadata: agentmodel.AgentRemoteConfigMetadata{Namespace: "default", Name: "config"},
				}
			},
			configs.PutAgentRemoteConfig,
			func(ctx context.Context) (*agentmodel.AgentRemoteConfig, error) {
				return configs.GetAgentRemoteConfig(ctx, "default", "config", nil)
			},
			func(v *agentmodel.AgentRemoteConfig) int64 { return v.Metadata.ResourceVersion },
			func(v *agentmodel.AgentRemoteConfig) { v.MarkDeleted(time.Now(), "test") })
	})
	t.Run("namespace", func(t *testing.T) {
		checkVersionedWrites(t, func() *agentmodel.Namespace { return agentmodel.NewNamespace("team") },
			namespaces.PutNamespace,
			func(ctx context.Context) (*agentmodel.Namespace, error) {
				return namespaces.GetNamespace(ctx, "team", nil)
			},
			func(v *agentmodel.Namespace) int64 { return v.Metadata.ResourceVersion },
			func(v *agentmodel.Namespace) { v.MarkAsDeleted(time.Now(), "test") })
	})
}

func checkVersionedWrites[T any](t *testing.T, fresh func() T,
	put func(context.Context, T) (T, error), get func(context.Context) (T, error),
	version func(T) int64, markDeleted func(T),
) {
	t.Helper()
	ctx := t.Context()

	const writers = 8

	start := make(chan struct{})
	results := make(chan error, writers)

	var wg sync.WaitGroup
	for range writers {
		wg.Go(func() { <-start; _, err := put(ctx, fresh()); results <- err })
	}

	close(start)
	wg.Wait()
	close(results)

	successes := 0

	for err := range results {
		if err == nil {
			successes++
		} else {
			require.ErrorIs(t, err, model.ErrConflict)
		}
	}

	require.Equal(t, 1, successes, "only one concurrent creator may insert")

	current, err := get(ctx)
	require.NoError(t, err)
	stale, err := get(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, version(current))
	updated, err := put(ctx, current)
	require.NoError(t, err)
	require.Equal(t, version(stale)+1, version(updated))
	_, err = put(ctx, stale)
	require.ErrorIs(t, err, model.ErrConflict)
	require.EqualValues(t, 1, version(stale), "failed writes must not mutate the caller's token")
	// A stale delete must not tombstone a newer version.
	markDeleted(stale)
	_, err = put(ctx, stale)
	require.ErrorIs(t, err, model.ErrConflict)
	current, err = get(ctx)
	require.NoError(t, err)
	markDeleted(current)
	deleted, err := put(ctx, current)
	require.NoError(t, err)
	require.Equal(t, version(updated)+1, version(deleted))

	_, err = get(ctx)
	require.ErrorIs(t, err, model.ErrResourceNotExist)
	_, err = put(ctx, fresh())
	require.ErrorIs(t, err, model.ErrConflict, "tombstones reserve their logical key")
}
