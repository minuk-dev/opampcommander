package agentmodel_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// TestSelectableFieldsMatchProjections is the guard that keeps a documented
// allowlist and the projection it describes from drifting apart: a field
// advertised but never projected would silently match nothing, and a field
// projected but not advertised would be rejected with a 400 despite working.
func TestSelectableFieldsMatchProjections(t *testing.T) {
	t.Parallel()

	agent := agentmodel.NewAgent(uuid.New())

	testCases := []struct {
		name    string
		allowed []string
		values  model.SelectorValues
	}{
		{"agent", agentmodel.AgentSelectableFields, agent.SelectorValuesAt(time.Now())},
		{"agentgroup", agentmodel.AgentGroupSelectableFields, (&agentmodel.AgentGroup{}).SelectorValues()},
		{"agentpackage", agentmodel.AgentPackageSelectableFields, (&agentmodel.AgentPackage{}).SelectorValues()},
		{
			"agentremoteconfig",
			agentmodel.AgentRemoteConfigSelectableFields,
			(&agentmodel.AgentRemoteConfig{}).SelectorValues(),
		},
		{"certificate", agentmodel.CertificateSelectableFields, (&agentmodel.Certificate{}).SelectorValues()},
		{"container", agentmodel.ContainerSelectableFields, (&agentmodel.Container{}).SelectorValues()},
		{"endpoint", agentmodel.EndpointSelectableFields, (&agentmodel.Endpoint{}).SelectorValues()},
		{"host", agentmodel.HostSelectableFields, (&agentmodel.Host{}).SelectorValues()},
		{"namespace", agentmodel.NamespaceSelectableFields, (&agentmodel.Namespace{}).SelectorValues()},
		{
			"remoteconfigschema",
			agentmodel.RemoteConfigSchemaSelectableFields,
			(&agentmodel.RemoteConfigSchema{}).SelectorValues(),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			projected := make([]string, 0, len(testCase.values.Fields))
			for field := range testCase.values.Fields {
				projected = append(projected, field)
			}

			assert.ElementsMatch(t, testCase.allowed, projected)
		})
	}
}

func TestAgentSelectorValuesAt_Connected(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.September, 2, 12, 0, 0, 0, time.UTC)

	agent := agentmodel.NewAgent(uuid.New())
	agent.Status.Connected = true
	agent.Status.LastReportedAt = now.Add(-time.Second)

	assert.Equal(t, "true", agent.SelectorValuesAt(now).Fields["status.connected"])

	// Past the staleness window the flag alone must not read as connected — the
	// same rule IsConnectedAt applies, so a filtered list and the badge agree.
	stale := now.Add(agentmodel.DefaultConnectionStaleness + time.Second)
	assert.Equal(t, "false", agent.SelectorValuesAt(stale).Fields["status.connected"])
}

func TestAgentSelectorValuesAt_LabelsAreIdentifyingAttributes(t *testing.T) {
	t.Parallel()

	agent := agentmodel.NewAgent(uuid.New())
	agent.Metadata.Description.IdentifyingAttributes = map[string]string{"service.namespace": "payments"}

	values := agent.SelectorValuesAt(time.Now())

	assert.Equal(t, map[string]string{"service.namespace": "payments"}, values.Labels)
	require.NotEmpty(t, values.Name, "an agent's name for prefix search is its instance UID")
	assert.Equal(t, agent.Metadata.InstanceUID.String(), values.Name)
}
