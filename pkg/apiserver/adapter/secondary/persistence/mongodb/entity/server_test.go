package entity_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/mongodb/entity"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

func TestServerEntity_RoundTrip(t *testing.T) {
	t.Parallel()

	// bson.DateTime carries millisecond precision, so use a millisecond-aligned UTC
	// time to keep the condition timestamp stable across the round-trip.
	heartbeatAt := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	transitionAt := heartbeatAt.Add(-time.Minute)

	domainServer := &agentmodel.Server{
		ID:              "server-1",
		Address:         "10.0.0.5:8081",
		LastHeartbeatAt: heartbeatAt,
		Conditions: []model.Condition{
			{
				Type:               model.ConditionTypeAlive,
				Status:             model.ConditionStatusTrue,
				LastTransitionTime: transitionAt,
				Reason:             "heartbeat",
				Message:            "Server is alive",
			},
		},
	}

	got := entity.ToServerEntity(domainServer).ToDomainModel()

	require.NotNil(t, got)
	assert.Equal(t, domainServer.ID, got.ID)
	assert.Equal(t, domainServer.Address, got.Address)
	assert.True(t, domainServer.LastHeartbeatAt.Equal(got.LastHeartbeatAt))
	require.Len(t, got.Conditions, 1)
	assert.Equal(t, model.ConditionTypeAlive, got.Conditions[0].Type)
	assert.Equal(t, model.ConditionStatusTrue, got.Conditions[0].Status)
	assert.Equal(t, "heartbeat", got.Conditions[0].Reason)
	assert.Equal(t, "Server is alive", got.Conditions[0].Message)
	assert.True(t, transitionAt.Equal(got.Conditions[0].LastTransitionTime))
}

func TestServerEntity_NilHandling(t *testing.T) {
	t.Parallel()

	assert.Nil(t, entity.ToServerEntity(nil))

	var nilEntity *entity.Server
	assert.Nil(t, nilEntity.ToDomainModel())
}
