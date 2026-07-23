package agentservice_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// apFakePersistence is a minimal in-memory AgentPackagePersistencePort for the
// lifecycle tests. It records the last Put and serves a single stored package.
type apFakePersistence struct {
	stored   *agentmodel.AgentPackage
	getErr   error
	putCalls int
	lastPut  *agentmodel.AgentPackage
}

func (f *apFakePersistence) GetAgentPackage(
	_ context.Context, _ string, _ string, _ *model.GetOptions,
) (*agentmodel.AgentPackage, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}

	if f.stored == nil {
		return nil, model.ErrResourceNotExist
	}

	return f.stored, nil
}

func (f *apFakePersistence) PutAgentPackage(
	_ context.Context, agentPackage *agentmodel.AgentPackage,
) (*agentmodel.AgentPackage, error) {
	f.putCalls++
	f.lastPut = agentPackage

	return agentPackage, nil
}

func (f *apFakePersistence) ListAgentPackages(
	_ context.Context, _ *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentPackage], error) {
	return &model.ListResponse[*agentmodel.AgentPackage]{}, nil
}

var _ agentport.AgentPackagePersistencePort = (*apFakePersistence)(nil)

func TestAgentPackageService_CreateAgentPackage_Stamps(t *testing.T) {
	t.Parallel()

	persistence := &apFakePersistence{}
	svc := agentservice.NewAgentPackageService(persistence)

	input := &agentmodel.AgentPackage{
		Metadata: agentmodel.AgentPackageMetadata{Name: "pkg", Namespace: "default"},
		Spec:     agentmodel.AgentPackageSpec{Version: "1.0.0"},
	}

	created, err := svc.CreateAgentPackage(t.Context(), input, "tester")

	require.NoError(t, err)
	assert.Equal(t, 1, persistence.putCalls)
	require.NotEmpty(t, created.Status.Conditions, "creation must record a condition")

	cond := created.Status.Conditions[0]
	assert.Equal(t, model.ConditionTypeCreated, cond.Type)
	assert.Equal(t, "tester", cond.Reason, "the acting user must be stamped as the condition reason")
}

func TestAgentPackageService_CreateAgentPackage_RejectsExisting(t *testing.T) {
	t.Parallel()

	stored := &agentmodel.AgentPackage{
		Metadata: agentmodel.AgentPackageMetadata{Name: "pkg", Namespace: "default", ResourceVersion: 5},
		Spec:     agentmodel.AgentPackageSpec{Version: "1.0.0"},
	}
	persistence := &apFakePersistence{stored: stored}
	svc := agentservice.NewAgentPackageService(persistence)

	input := &agentmodel.AgentPackage{
		Metadata: agentmodel.AgentPackageMetadata{Name: "pkg", Namespace: "default"},
		Spec:     agentmodel.AgentPackageSpec{Version: "2.0.0"},
	}

	_, err := svc.CreateAgentPackage(t.Context(), input, "tester")

	// Creating over an existing package must be rejected, not silently upsert it
	// (which would overwrite the package and rewind its ResourceVersion).
	require.ErrorIs(t, err, model.ErrResourceAlreadyExist)
	assert.Equal(t, 0, persistence.putCalls, "no write may happen when the package already exists")
}

func TestAgentPackageService_GetListSaveDelete(t *testing.T) {
	t.Parallel()

	t.Run("get delegates to persistence", func(t *testing.T) {
		t.Parallel()

		stored := &agentmodel.AgentPackage{
			Metadata: agentmodel.AgentPackageMetadata{Name: "pkg", Namespace: "default"},
		}
		svc := agentservice.NewAgentPackageService(&apFakePersistence{stored: stored})

		got, err := svc.GetAgentPackage(t.Context(), "default", "pkg", nil)
		require.NoError(t, err)
		assert.Equal(t, "pkg", got.Metadata.Name)
	})

	t.Run("get propagates not-found", func(t *testing.T) {
		t.Parallel()

		svc := agentservice.NewAgentPackageService(&apFakePersistence{})

		_, err := svc.GetAgentPackage(t.Context(), "default", "missing", nil)
		require.ErrorIs(t, err, model.ErrResourceNotExist)
	})

	t.Run("list delegates to persistence", func(t *testing.T) {
		t.Parallel()

		svc := agentservice.NewAgentPackageService(&apFakePersistence{})

		resp, err := svc.ListAgentPackages(t.Context(), nil)
		require.NoError(t, err)
		assert.Empty(t, resp.Items)
	})

	t.Run("save writes as-is", func(t *testing.T) {
		t.Parallel()

		persistence := &apFakePersistence{}
		svc := agentservice.NewAgentPackageService(persistence)
		pkg := &agentmodel.AgentPackage{
			Metadata: agentmodel.AgentPackageMetadata{Name: "pkg", Namespace: "default", ResourceVersion: 2},
		}

		_, err := svc.SaveAgentPackage(t.Context(), pkg)
		require.NoError(t, err)
		assert.Equal(t, 1, persistence.putCalls)
		assert.Equal(t, int64(2), persistence.lastPut.Metadata.ResourceVersion)
	})

	t.Run("delete stamps a deletion and persists it", func(t *testing.T) {
		t.Parallel()

		stored := &agentmodel.AgentPackage{
			Metadata: agentmodel.AgentPackageMetadata{Name: "pkg", Namespace: "default"},
		}
		persistence := &apFakePersistence{stored: stored}
		svc := agentservice.NewAgentPackageService(persistence)

		deletedAt := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
		require.NoError(t, svc.DeleteAgentPackage(t.Context(), "default", "pkg", deletedAt, "tester"))

		require.Equal(t, 1, persistence.putCalls)
		require.NotNil(t, persistence.lastPut.Metadata.DeletedAt)
		assert.Equal(t, deletedAt, *persistence.lastPut.Metadata.DeletedAt)
	})
}

func TestAgentPackageService_UpdateAgentPackage_PreservesImmutableFields(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	stored := &agentmodel.AgentPackage{
		Metadata: agentmodel.AgentPackageMetadata{Name: "pkg", Namespace: "default", CreatedAt: createdAt},
		Spec:     agentmodel.AgentPackageSpec{Version: "1.0.0"},
		Status:   agentmodel.AgentPackageStatus{Conditions: []model.Condition{{Type: model.ConditionTypeCreated}}},
	}

	persistence := &apFakePersistence{stored: stored}
	svc := agentservice.NewAgentPackageService(persistence)

	incoming := &agentmodel.AgentPackage{
		Metadata: agentmodel.AgentPackageMetadata{
			Name:      "pkg",
			Namespace: "default",
			CreatedAt: time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		Spec: agentmodel.AgentPackageSpec{Version: "2.0.0"},
	}

	updated, err := svc.UpdateAgentPackage(t.Context(), "default", "pkg", incoming)

	require.NoError(t, err)
	assert.Equal(t, createdAt, updated.Metadata.CreatedAt, "CreatedAt must be preserved from the stored package")
	assert.Equal(t, "2.0.0", updated.Spec.Version, "mutable spec must be applied")
	assert.NotEmpty(t, updated.Status.Conditions, "existing lifecycle conditions must be preserved")
}
