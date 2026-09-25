package testutil

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
)

// EventuallyAgent polls the agent through apiClient until condition holds and returns
// the last agent it observed. On timeout it fails the test and dumps that agent, so a
// failure shows what the server actually reported instead of just "never satisfied".
func EventuallyAgent(
	tb testing.TB,
	apiClient *client.Client,
	namespace string,
	uid uuid.UUID,
	condition func(*v1.Agent) bool,
	waitFor, tick time.Duration,
	msg string,
) *v1.Agent {
	tb.Helper()

	var last *v1.Agent

	satisfied := assert.Eventually(tb, func() bool {
		agent, err := apiClient.AgentService.GetAgent(tb.Context(), namespace, uid)
		if err != nil {
			return false
		}

		last = agent

		return condition(agent)
	}, waitFor, tick, msg)
	if !satisfied {
		tb.Fatalf("last observed agent:\n%s", DumpJSON(tb, last))
	}

	return last
}

// DumpJSON renders value as indented JSON for failure messages.
func DumpJSON(tb testing.TB, value any) string {
	tb.Helper()

	out, err := json.MarshalIndent(value, "", "  ")
	require.NoError(tb, err)

	return string(out)
}
