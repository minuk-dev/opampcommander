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
		}, nil, nil, nil, map[string]agentmodel.AgentOtherConnectionSettings{},
	))
	// Simulate a group overriding the effective offer without changing the independent fallback.
	groupInfo, err := agentmodel.NewConnectionInfo(&agentmodel.AgentOpAMPConnectionSettings{
		DestinationEndpoint: "wss://group.test", Certificate: &agentmodel.AgentCertificate{Cert: []byte("group-cert")},
	}, nil, nil, nil, nil)
	require.NoError(t, err)

	agent.Spec.ConnectionInfo = groupInfo
	agent.Status.ConnectionSettingsStatus = agentmodel.AgentConnectionSettingsStatus{
		LastConnectionSettingsHash: agent.Spec.ConnectionInfo.Hash.Bytes(),
		Status:                     agentmodel.ConnectionSettingsStatusApplied,
	}
	agent.Status.ActiveClientCertificateHash = []byte("active-certificate-fingerprint")

	data, err := bson.Marshal(entity.AgentFromDomain(agent))
	require.NoError(t, err)

	var stored entity.Agent
	require.NoError(t, bson.Unmarshal(data, &stored))
	reloaded := stored.ToDomain()
	require.Equal(t, agent.Spec.ConnectionInfo.Hash, reloaded.Spec.ConnectionInfo.Hash)
	require.Equal(t, []byte("group-cert"), reloaded.Spec.ConnectionInfo.OpAMP().Certificate.Cert)
	require.Equal(t, agent.Spec.PerAgentConnectionInfo.Hash, reloaded.Spec.PerAgentConnectionInfo.Hash)
	require.Equal(t, []byte("new-cert"), reloaded.Spec.PerAgentConnectionInfo.OpAMP().Certificate.Cert)
	clone := reloaded.Clone()
	clone.Spec.PerAgentConnectionInfo.OpAMP().Certificate.Cert[0] = 'X'
	require.Equal(t, []byte("new-cert"), reloaded.Spec.PerAgentConnectionInfo.OpAMP().Certificate.Cert)
	require.Equal(t, agent.Status.ConnectionSettingsStatus, reloaded.Status.ConnectionSettingsStatus)
	require.Equal(t, agent.Status.ActiveClientCertificateHash, reloaded.Status.ActiveClientCertificateHash)
}
