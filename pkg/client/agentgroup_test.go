package client_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/client"
)

func TestListAgentGroupsByAgentOptions(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/namespaces/team/agents/collector/agentgroups", r.URL.Path)

		for key, value := range map[string]string{
			"limit": "50", "continue": "cursor", "includeDeleted": "true",
			"labelSelector": "env=prod", "fieldSelector": "metadata.namespace=team",
			"name": "group", "nameContains": "otel",
		} {
			assert.Equal(t, value, r.URL.Query().Get(key), key)
		}

		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"items":[],"metadata":{"continue":"next","remainingItemCount":100001}}`))
		assert.NoError(t, err)
	}))
	defer server.Close()

	cli := client.New(server.URL)
	response, err := cli.AgentGroupService.ListAgentGroupsByAgent(t.Context(), "team", "collector",
		client.WithLimit(50), client.WithContinueToken("cursor"), client.WithIncludeDeleted(true),
		client.WithLabelSelector("env=prod"), client.WithFieldSelector("metadata.namespace=team"),
		client.WithName("group"), client.WithNameContains("otel"))
	require.NoError(t, err)
	assert.Equal(t, "next", response.Metadata.Continue)
	assert.EqualValues(t, 100001, response.Metadata.RemainingItemCount)
}
