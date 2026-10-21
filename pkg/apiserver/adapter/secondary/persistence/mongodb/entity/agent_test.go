package entity_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/mongodb/entity"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
)

func TestAgent_RestartRequestSurvivesPersistence(t *testing.T) {
	t.Parallel()

	agent := agentmodel.NewAgent(uuid.New())
	agent.Status.ComponentHealth.StartTime = time.Date(2026, time.October, 13, 0, 0, 0, 0, time.UTC)
	requested := agent.Status.ComponentHealth.StartTime.Add(time.Minute)
	agent.Spec.RestartInfo = &agentmodel.AgentRestartInfo{RequiredRestartedAt: requested}

	stored := entity.AgentFromDomain(agent)
	encoded, err := bson.Marshal(stored)
	require.NoError(t, err)

	var decoded entity.Agent
	require.NoError(t, bson.Unmarshal(encoded, &decoded))
	reloaded := decoded.ToDomain()
	require.WithinDuration(t, requested, reloaded.Spec.RestartInfo.RequiredRestartedAt, 0)
	require.True(t, reloaded.ShouldBeRestarted(), "heartbeat recovery must still offer the persisted restart")
	reloaded.Status.ComponentHealth.StartTime = requested.Add(time.Second)
	require.False(t, reloaded.ShouldBeRestarted(), "an acknowledged restart must not repeat")
}

func TestAgent_NoRestartRequest(t *testing.T) {
	t.Parallel()

	for _, storedTime := range []bson.DateTime{0, bson.NewDateTimeFromTime(time.Time{})} {
		spec := (&entity.AgentSpec{RequiredRestartedAt: storedTime}).ToDomain()
		require.True(t, spec.RestartInfo.RequiredRestartedAt.IsZero())
		require.False(t, spec.RestartInfo.ShouldBeRestarted(time.Time{}),
			"a missing request must not restart an agent with an unknown start time")
	}
}

func TestAgent_RemoteConfigFailureSurvivesBSON(t *testing.T) {
	t.Parallel()

	agent := agentmodel.NewAgent(uuid.New())

	now := time.Now().UTC().Truncate(time.Millisecond)
	require.NoError(t, agent.ReportRemoteConfigStatus(&agentmodel.AgentRemoteConfigStatus{
		LastRemoteConfigHash: []byte("bad"), Status: agentmodel.RemoteConfigStatusFailed,
		ErrorMessage: "invalid collector config", LastUpdatedAt: now,
	}))
	require.NoError(t, agent.ReportRemoteConfigStatus(&agentmodel.AgentRemoteConfigStatus{
		LastRemoteConfigHash: []byte("good"), Status: agentmodel.RemoteConfigStatusApplied,
		LastUpdatedAt: now.Add(time.Second),
	}))
	encoded, err := bson.Marshal(entity.AgentFromDomain(agent))
	require.NoError(t, err)

	var stored entity.Agent
	require.NoError(t, bson.Unmarshal(encoded, &stored))
	require.Equal(t, agent.Status.RemoteConfigStatus, stored.ToDomain().Status.RemoteConfigStatus)
	clone := agent.Clone()
	clone.Status.RemoteConfigStatus.LastFailedHash[0] = 'x'
	require.Equal(t, []byte("bad"), agent.Status.RemoteConfigStatus.LastFailedHash)
}
