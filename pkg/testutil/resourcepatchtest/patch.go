// Package resourcepatchtest shares PATCH contract checks between storage adapters.
package resourcepatchtest

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	groupsvc "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/agentgroup"
	packagesvc "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/agentpackage"
	configsvc "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/agentremoteconfig"
	namespacesvc "github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/namespace"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

// CheckResourcePatches checks the application contract against each storage adapter.
func CheckResourcePatches(t *testing.T, groups agentport.AgentGroupPersistencePort,
	packages agentport.AgentPackagePersistencePort, configs agentport.AgentRemoteConfigPersistencePort,
	namespaces agentport.NamespacePersistencePort,
) {
	t.Helper()
	logger := testutil.NewBase(t).Logger
	ctx := t.Context()
	t.Run("agentgroup", func(t *testing.T) {
		_, err := groups.PutAgentGroup(ctx, "default", "patch-group", agentmodel.NewAgentGroup("default",
			"patch-group", nil, time.Now(), "test"))
		require.NoError(t, err)

		queue := &patchChangeStore{}
		svc := groupsvc.NewManageService(agentservice.NewAgentGroupService(groups, configs, nil, nil, nil, queue,
			logger), nil, logger)

		checkPatches(t, "attributes", func(data []byte) (any, error) {
			return svc.PatchAgentGroup(ctx, "default",
				"patch-group", data)
		},
			func() error {
				current, err := groups.GetAgentGroup(ctx, "default", "patch-group", nil)
				require.NoError(t, err)
				current.MarkDeleted(time.Now(), "test")
				_, err = groups.PutAgentGroup(ctx, "default", "patch-group", current)

				return err
			})
		require.Equal(t, 3, queue.writes)

		_, err = svc.PatchAgentGroup(ctx, "default", "missing", []byte(`{}`))
		require.ErrorIs(t, err, model.ErrResourceNotExist)
	})
	t.Run("agentpackage", func(t *testing.T) {
		_, err := packages.PutAgentPackage(ctx,
			&agentmodel.AgentPackage{Metadata: agentmodel.AgentPackageMetadata{Namespace: "default",
				Name: "patch-package"}})
		require.NoError(t, err)

		svc := packagesvc.NewAgentPackageService(agentservice.NewAgentPackageService(packages), logger)

		checkPatches(t, "attributes", func(data []byte) (any, error) {
			return svc.PatchAgentPackage(ctx, "default",
				"patch-package", data)
		},
			func() error {
				current, err := packages.GetAgentPackage(ctx, "default", "patch-package", nil)
				require.NoError(t, err)
				current.MarkAsDeleted(time.Now(), "test")
				_, err = packages.PutAgentPackage(ctx, current)

				return err
			})

		_, err = svc.PatchAgentPackage(ctx, "default", "missing", []byte(`{}`))
		require.ErrorIs(t, err, model.ErrResourceNotExist)
	})
	t.Run("agentremoteconfig", func(t *testing.T) {
		_, err := configs.PutAgentRemoteConfig(ctx,
			&agentmodel.AgentRemoteConfig{Metadata: agentmodel.AgentRemoteConfigMetadata{Namespace: "default",
				Name: "patch-config"}})
		require.NoError(t, err)

		effects := &patchEffects{events: make(chan string, 20)}
		svc := configsvc.NewAgentRemoteConfigService(agentservice.NewAgentRemoteConfigService(configs, nil, nil,
			nil, logger), effects, effects, logger)

		checkPatches(t, "attributes", func(data []byte) (any, error) {
			return svc.PatchAgentRemoteConfig(ctx, "default", "patch-config", data)
		},
			func() error {
				current, err := configs.GetAgentRemoteConfig(ctx, "default", "patch-config", nil)
				require.NoError(t, err)
				current.MarkDeleted(time.Now(), "test")
				_, err = configs.PutAgentRemoteConfig(ctx, current)

				return err
			})

		for range 6 {
			select {
			case <-effects.events:
			case <-time.After(time.Second):
				t.Fatal("missing propagation")
			}
		}

		select {
		case event := <-effects.events:
			t.Fatalf("duplicate propagation: %s", event)
		case <-time.After(20 * time.Millisecond):
		}

		_, err = svc.PatchAgentRemoteConfig(ctx, "default", "missing", []byte(`{}`))
		require.ErrorIs(t, err, model.ErrResourceNotExist)
	})
	t.Run("namespace", func(t *testing.T) {
		_, err := namespaces.PutNamespace(ctx,
			&agentmodel.Namespace{Metadata: agentmodel.NamespaceMetadata{Name: "patch-namespace"}})
		require.NoError(t, err)

		svc := namespacesvc.NewNamespaceService(agentservice.NewNamespaceService(namespaces, nil, nil, nil, nil,
			nil, ""), logger)

		checkPatches(t, "labels", func(data []byte) (any, error) {
			return svc.PatchNamespace(ctx, "patch-namespace",
				data)
		},
			func() error {
				current, err := namespaces.GetNamespace(ctx, "patch-namespace", nil)
				require.NoError(t, err)
				current.MarkAsDeleted(time.Now(), "test")
				_, err = namespaces.PutNamespace(ctx, current)

				return err
			})

		_, err = svc.PatchNamespace(ctx, "missing", []byte(`{}`))
		require.ErrorIs(t, err, model.ErrResourceNotExist)
		_, err = namespaces.PutNamespace(ctx,
			&agentmodel.Namespace{Metadata: agentmodel.NamespaceMetadata{Name: "concurrent-patch"}})
		require.NoError(t, err)

		var wg sync.WaitGroup
		for _, patch := range []string{`{"metadata":{"labels":{"a":"one"}}}`,
			`{"metadata":{"annotations":{"b":"two"}}}`} {
			wg.Go(func() {
				_, err := svc.PatchNamespace(ctx, "concurrent-patch", []byte(patch))
				if err != nil {
					t.Error(err)
				}
			})
		}

		wg.Wait()

		current, err := svc.GetNamespace(ctx, "concurrent-patch", nil)
		require.NoError(t, err)
		require.Equal(t, "one", current.Metadata.Labels["a"])
		require.Equal(t, "two", current.Metadata.Annotations["b"])
	})
}

func checkPatches(t *testing.T, field string, patch func([]byte) (any, error), tombstone func() error) {
	t.Helper()

	apply := func(body string, version int64) map[string]string {
		t.Helper()

		result, err := patch([]byte(body))
		require.NoError(t, err)
		data, err := json.Marshal(result)
		require.NoError(t, err)

		var decoded struct {
			Metadata struct {
				ResourceVersion int64             `json:"resourceVersion,string"`
				Attributes      map[string]string `json:"attributes"`
				Labels          map[string]string `json:"labels"`
			} `json:"metadata"`
		}
		require.NoError(t, json.Unmarshal(data, &decoded))
		require.Equal(t, version, decoded.Metadata.ResourceVersion)

		if field == "labels" {
			return decoded.Metadata.Labels
		}

		return decoded.Metadata.Attributes
	}
	body := fmt.Sprintf(`{"metadata":{"resourceVersion":"1","%s":{"keep":"yes","remove":"old"}}}`, field)
	apply(body, 2)
	apply(body, 2) // Lost response: same desired state, original revision.
	apply(`{}`, 2)

	_, err := patch([]byte(fmt.Sprintf(`{"metadata":{"resourceVersion":"1","%s":{"keep":"stale"}}}`, field)))
	require.ErrorIs(t, err, model.ErrConflict)

	values := apply(fmt.Sprintf(`{"metadata":{"%s":{"remove":null,"new":"value"}}}`, field), 3)
	require.Equal(t, map[string]string{"keep": "yes", "new": "value"}, values)
	values = apply(fmt.Sprintf(`{"metadata":{"%s":{"keep":"last"}}}`, field), 4)
	require.Equal(t, "last", values["keep"])

	for _, invalid := range []string{`{"status":null}`, `{"metadata":{"name":"different"}}`,
		`{"metadata":{"resourceVersion":null}}`} {
		_, err = patch([]byte(invalid))
		require.ErrorIs(t, err, model.ErrInvalidArgument)
		apply(`{}`, 4)
	}

	require.NoError(t, tombstone())

	_, err = patch([]byte(`{}`))
	require.ErrorIs(t, err, model.ErrResourceNotExist)
}

type patchChangeStore struct {
	agentport.AgentGroupChangeStore

	writes int
}

func (s *patchChangeStore) Enqueue(context.Context, agentport.AgentGroupChange) error {
	s.writes++

	return nil
}

type patchEffects struct {
	agentport.AgentGroupUsecase
	agentport.EndpointDetectionUsecase

	events chan string
}

func (s *patchEffects) PropagateAgentRemoteConfigChange(context.Context, string, string) error {
	s.events <- "group"

	return nil
}
func (s *patchEffects) ReconcileEndpointsFromRemoteConfig(context.Context, *agentmodel.AgentRemoteConfig) error {
	s.events <- "endpoint"

	return nil
}
