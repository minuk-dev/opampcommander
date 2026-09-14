package client_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/client"
)

// newQueryRecorder serves an empty listing and records the query of the request
// it answered, so a test can assert on the wire form of the list options.
func newQueryRecorder(t *testing.T) (*client.Client, *url.Values) {
	t.Helper()

	var recorded url.Values

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorded = r.URL.Query()

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[],"continue":"","remainingItemCount":0}`))
	}))
	t.Cleanup(server.Close)

	return client.New(server.URL), &recorded
}

func TestListOptions_SelectorsAreSentAsQueryParams(t *testing.T) {
	t.Parallel()

	cli, recorded := newQueryRecorder(t)

	_, err := cli.AgentService.ListAgents(t.Context(), "default",
		client.WithLabelSelector("env=prod,tier notin (canary,dev)"),
		client.WithFieldSelector("status.connected=true"),
		client.WithName("otel-"),
	)
	require.NoError(t, err)

	assert.Equal(t, "env=prod,tier notin (canary,dev)", recorded.Get("labelSelector"))
	assert.Equal(t, "status.connected=true", recorded.Get("fieldSelector"))
	assert.Equal(t, "otel-", recorded.Get("name"))
}

// The two metadata selectors are separate parameters, because they read different
// metadata: one an operator set, one the resource reported. Sending the wrong one
// is the server's 400, not something the client papers over.
func TestListOptions_AttributeSelectorIsItsOwnParameter(t *testing.T) {
	t.Parallel()

	cli, recorded := newQueryRecorder(t)

	_, err := cli.AgentService.ListAgents(t.Context(), "default",
		client.WithAttributeSelector("service.name=otel-collector,os.type=linux"))
	require.NoError(t, err)

	assert.Equal(t, "service.name=otel-collector,os.type=linux", recorded.Get("attributeSelector"))
	assert.False(t, recorded.Has("labelSelector"))
}

func TestListOptions_EmptySelectorsAreOmitted(t *testing.T) {
	t.Parallel()

	cli, recorded := newQueryRecorder(t)

	_, err := cli.AgentService.ListAgents(t.Context(), "default",
		client.WithLabelSelector(""),
		client.WithAttributeSelector(""),
		client.WithFieldSelector(""),
		client.WithName(""),
	)
	require.NoError(t, err)

	assert.False(t, recorded.Has("labelSelector"))
	assert.False(t, recorded.Has("attributeSelector"))
	assert.False(t, recorded.Has("fieldSelector"))
	assert.False(t, recorded.Has("name"))
}

// The selectors have to survive alongside the older options rather than replace
// them on the wire: a caller may still pass --non-identifying-selector.
func TestListOptions_SelectorsCombineWithLegacyOptions(t *testing.T) {
	t.Parallel()

	cli, recorded := newQueryRecorder(t)

	_, err := cli.AgentService.ListAgents(t.Context(), "default",
		client.WithLabelSelector("env=prod"),
		client.WithNonIdentifyingSelector(map[string]string{"os.type": "linux"}),
		client.WithConnectedOnly(true),
		client.WithLimit(50),
	)
	require.NoError(t, err)

	assert.Equal(t, "env=prod", recorded.Get("labelSelector"))
	assert.Equal(t, []string{"os.type=linux"}, (*recorded)["nonIdentifyingSelector"])
	assert.Equal(t, "true", recorded.Get("connected"))
	assert.Equal(t, "50", recorded.Get("limit"))
}
