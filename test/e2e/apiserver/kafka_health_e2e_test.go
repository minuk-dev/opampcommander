//go:build e2e

package apiserver_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"

	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

func TestE2E_APIServer_KafkaHealthSurvivesBrokerOutage(t *testing.T) {
	t.Parallel()
	testcontainers.SkipIfProviderIsNotHealthy(t)

	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	base := testutil.NewBase(t)
	mongodb := base.StartMongoDB()
	broker := base.StartKafka()
	server := base.StartAPIServerWithKafka(mongodb.URI, broker.Broker, "opampcommander_kafka_health")
	server.WaitForReady()

	client := &http.Client{Timeout: 3 * time.Second}

	healthReason := func() string {
		t.Helper()

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.ManagementEndpoint+"/healthz", nil)
		require.NoError(t, err)
		response, err := client.Do(req)
		require.NoError(t, err)

		defer func() { require.NoError(t, response.Body.Close()) }()

		require.Equal(t, http.StatusOK, response.StatusCode)

		var reasons map[string]string
		require.NoError(t, json.NewDecoder(response.Body).Decode(&reasons))
		t.Logf("Kafka health reason: %q", reasons["Kafka"])

		return reasons["Kafka"]
	}

	require.Empty(t, healthReason())
	require.NoError(t, broker.Stop(t.Context(), nil))
	require.Eventually(t, func() bool {
		return healthReason() != ""
	}, 20*time.Second, time.Second, "broker loss must appear after the cached success expires")
	require.True(t, server.IsReady(), "broker loss must not drain the server")

	require.NoError(t, broker.Start(t.Context()))

	restartedAddress, err := broker.PortEndpoint(t.Context(), "9093/tcp", "")
	require.NoError(t, err)
	require.Equal(t, broker.Broker, restartedAddress, "restarting Kafka must preserve its configured address")

	require.Eventually(t, func() bool {
		return healthReason() == ""
	}, time.Minute, time.Second, "health must recover after the broker restarts")
	require.True(t, server.IsReady())
}
