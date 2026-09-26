package entity_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/mongodb/entity"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
)

func TestAgentConnectionOfferAndStatusRoundTrip(t *testing.T) {
	t.Parallel()

	agent := agentmodel.NewAgent(uuid.New())
	require.NoError(t, agent.ApplyConnectionSettings(
		&agentmodel.AgentOpAMPConnectionSettings{
			DestinationEndpoint: "wss://example.test/api/v1/opamp",
			Certificate:         &agentmodel.AgentCertificate{Cert: []byte("new-cert"), PrivateKey: []byte("new-key")},
		}, nil, nil, nil, nil,
	))
	agent.Status.ConnectionSettingsStatus = agentmodel.AgentConnectionSettingsStatus{
		LastConnectionSettingsHash: agent.Spec.ConnectionInfo.Hash.Bytes(),
		Status:                     agentmodel.ConnectionSettingsStatusApplied,
	}

	data, err := bson.Marshal(entity.AgentFromDomain(agent))
	require.NoError(t, err)

	var stored entity.Agent
	require.NoError(t, bson.Unmarshal(data, &stored))
	reloaded := stored.ToDomain()
	require.Equal(t, agent.Spec.ConnectionInfo.Hash, reloaded.Spec.ConnectionInfo.Hash)
	require.Equal(t, []byte("new-cert"), reloaded.Spec.ConnectionInfo.OpAMP().Certificate.Cert)
	require.Equal(t, agent.Status.ConnectionSettingsStatus, reloaded.Status.ConnectionSettingsStatus)
}
