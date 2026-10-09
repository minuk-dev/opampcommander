package agentmodel_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

func TestAgentPackage_MarkAsCreated(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC)
	pkg := &agentmodel.AgentPackage{
		Metadata: agentmodel.AgentPackageMetadata{Name: "pkg", Namespace: "default"},
		Spec:     agentmodel.AgentPackageSpec{Version: "1.0.0"},
		Status:   agentmodel.AgentPackageStatus{},
	}

	pkg.MarkAsCreated(now, "tester")

	assert.Equal(t, now, pkg.Metadata.CreatedAt)
	require.Len(t, pkg.Status.Conditions, 1)
	assert.Equal(t, model.ConditionTypeCreated, pkg.Status.Conditions[0].Type)
	assert.Equal(t, "tester", pkg.Status.Conditions[0].Reason)
}

func TestAgentPackage_ApplyUpdate_PreservesIdentityAndLifecycle(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	stored := &agentmodel.AgentPackage{
		Metadata: agentmodel.AgentPackageMetadata{
			Name:            "pkg",
			Namespace:       "default",
			ResourceVersion: 5,
			CreatedAt:       createdAt,
			Attributes:      agentmodel.Attributes{"team": "old"},
		},
		Spec:   agentmodel.AgentPackageSpec{Version: "1.0.0"},
		Status: agentmodel.AgentPackageStatus{Conditions: []model.Condition{{Type: model.ConditionTypeCreated}}},
	}

	incoming := &agentmodel.AgentPackage{
		Metadata: agentmodel.AgentPackageMetadata{
			Name:            "hacker",
			Namespace:       "other",
			ResourceVersion: 999,
			CreatedAt:       time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
			Attributes:      agentmodel.Attributes{"team": "new"},
		},
		Spec: agentmodel.AgentPackageSpec{Version: "2.0.0"},
	}

	stored.ApplyUpdate(incoming)

	// Mutable fields are applied.
	assert.Equal(t, "2.0.0", stored.Spec.Version)
	assert.Equal(t, agentmodel.Attributes{"team": "new"}, stored.Metadata.Attributes)

	// Immutable identity, lifecycle, and the optimistic-concurrency token are preserved.
	assert.Equal(t, "pkg", stored.Metadata.Name)
	assert.Equal(t, "default", stored.Metadata.Namespace)
	assert.Equal(t, createdAt, stored.Metadata.CreatedAt)
	assert.Equal(t, int64(5), stored.Metadata.ResourceVersion,
		"ApplyUpdate must not let a client-supplied model rewind the version")
	assert.Len(t, stored.Status.Conditions, 1)
}

func TestAgentPackage_MarkAsDeleted(t *testing.T) {
	t.Parallel()

	deletedAt := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
	pkg := &agentmodel.AgentPackage{
		Metadata: agentmodel.AgentPackageMetadata{Name: "pkg", Namespace: "default"},
		Spec:     agentmodel.AgentPackageSpec{Version: "1.0.0"},
		Status:   agentmodel.AgentPackageStatus{},
	}

	pkg.MarkAsDeleted(deletedAt, "tester")

	require.NotNil(t, pkg.Metadata.DeletedAt)
	assert.Equal(t, deletedAt, *pkg.Metadata.DeletedAt)
	require.Len(t, pkg.Status.Conditions, 1)
	assert.Equal(t, model.ConditionTypeDeleted, pkg.Status.Conditions[0].Type)
}

func TestDeclarativeSpecEquality(t *testing.T) {
	t.Parallel()

	left := agentmodel.AgentPackageSpec{}
	right := agentmodel.AgentPackageSpec{Headers: map[string]string{}, ContentHash: []byte{}}
	require.True(t, left.Equal(right))
	right.Version = "changed"
	require.False(t, left.Equal(right))

	remote := agentmodel.AgentRemoteConfigSpec{}
	require.True(t, remote.Equal(agentmodel.AgentRemoteConfigSpec{Value: []byte{}, SchemaRefs: []string{}}))
	require.False(t, remote.Equal(agentmodel.AgentRemoteConfigSpec{Value: []byte("changed")}))

	group := agentmodel.AgentGroupSpec{}
	require.True(t, group.Equal(agentmodel.AgentGroupSpec{
		Selector:           agentmodel.AgentSelector{IdentifyingAttributes: map[string]string{}},
		AgentRemoteConfigs: []agentmodel.AgentGroupAgentRemoteConfig{},
	}))
	require.False(t, group.Equal(agentmodel.AgentGroupSpec{Priority: 1}))

	group.AgentConnectionConfig = &agentmodel.AgentGroupConnectionConfig{
		OpAMPConnection: &agentmodel.OpAMPConnectionSettings{DestinationEndpoint: "wss://example.com"},
	}
	other := agentmodel.AgentGroupSpec{AgentConnectionConfig: &agentmodel.AgentGroupConnectionConfig{
		OpAMPConnection: &agentmodel.OpAMPConnectionSettings{
			DestinationEndpoint: "wss://example.com", Headers: map[string][]string{},
		},
		OtherConnections: map[string]agentmodel.OtherConnectionSettings{},
	}}
	require.True(t, group.Equal(other))
	other.AgentConnectionConfig.OpAMPConnection.DestinationEndpoint = "wss://changed.example.com"
	require.False(t, group.Equal(other))
}
