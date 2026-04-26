//go:build e2e

package apiserver_test

import (
	"context"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func listAgents(t *testing.T, baseURL string) []v1.Agent {
	t.Helper()

	c := client.New(baseURL, client.WithBasicAuth(testutil.DefaultAdminUsername, testutil.DefaultAdminPassword))
	resp, err := c.AgentService.ListAgents(t.Context(), "default")
	require.NoError(t, err)

	return resp.Items
}

func getAgentByID(t *testing.T, baseURL string, uid uuid.UUID) *v1.Agent {
	t.Helper()

	c := client.New(baseURL, client.WithBasicAuth(testutil.DefaultAdminUsername, testutil.DefaultAdminPassword))
	agent, err := c.AgentService.GetAgent(t.Context(), "default", uid)
	require.NoError(t, err)

	return agent
}

func tryGetAgentByIDWithClient(c *client.Client, uid uuid.UUID) (*v1.Agent, error) {
	return c.AgentService.GetAgent(context.Background(), "default", uid)
}

func findAgentByUID(agents []v1.Agent, uid uuid.UUID) *v1.Agent {
	idx := slices.IndexFunc(agents, func(a v1.Agent) bool {
		return a.Metadata.InstanceUID == uid
	})
	if idx < 0 {
		return nil
	}
	return &agents[idx]
}

