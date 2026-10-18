package kafka

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/management/healthcheck"
)

func TestKafkaHealthIndicator_CacheAndRecovery(t *testing.T) {
	t.Parallel()

	broker := sarama.NewMockBroker(t, 1)
	t.Cleanup(broker.Close)
	metadata := sarama.NewMockMetadataResponse(t).SetBroker(broker.Addr(), broker.BrokerID())
	broker.SetHandlerByMap(map[string]sarama.MockResponse{"MetadataRequest": metadata})
	indicator := NewHealthIndicator([]string{broker.Addr()})
	require.Equal(t, healthcheck.Health{Healthy: true}, indicator.Health(t.Context()))
	require.Len(t, broker.History(), 1)

	// An unresponsive broker is hidden only until the successful result expires.
	broker.SetHandlerByMap(nil)
	require.False(t, indicator.Health(t.Context()).Degraded)
	require.Len(t, broker.History(), 1)

	indicator.checkedAt = time.Now().Add(-healthCacheTTL)

	helper := healthcheck.NewHealthHelper([]healthcheck.HealthIndicator{indicator})
	response := httptest.NewRecorder()
	helper.IsHealth(response, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz", nil))
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), `"Kafka":"cross-server event delivery unavailable:`)
	require.True(t, indicator.Health(t.Context()).Degraded)
	require.Len(t, broker.History(), 2, "failed probes must be cached too")

	response = httptest.NewRecorder()
	helper.IsReady(response, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/readyz", nil))
	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{}`, response.Body.String())
	require.Len(t, broker.History(), 2, "readiness must not query Kafka")

	broker.SetHandlerByMap(map[string]sarama.MockResponse{"MetadataRequest": metadata})
	require.True(t, indicator.Health(t.Context()).Degraded)
	indicator.checkedAt = time.Now().Add(-healthCacheTTL)
	require.Equal(t, healthcheck.Health{Healthy: true}, indicator.Health(t.Context()))
	require.Len(t, broker.History(), 3)
}

func TestKafkaHealthIndicator_ConcurrentPolls(t *testing.T) {
	t.Parallel()

	broker := sarama.NewMockBroker(t, 1)
	t.Cleanup(broker.Close)
	broker.SetHandlerByMap(map[string]sarama.MockResponse{
		"MetadataRequest": sarama.NewMockMetadataResponse(t).SetBroker(broker.Addr(), broker.BrokerID()),
	})
	indicator := NewHealthIndicator([]string{broker.Addr()})

	var workers sync.WaitGroup
	for range 20 {
		workers.Go(func() {
			assert.Equal(t, healthcheck.Health{Healthy: true}, indicator.Health(t.Context()))
		})
	}

	workers.Wait()
	require.Len(t, broker.History(), 1)
}

func TestKafkaHealthIndicator_ProbeFailures(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name    string
		brokers []string
	}{
		{name: "no configured brokers"},
		{name: "unreachable broker", brokers: []string{"127.0.0.1:0"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			indicator := NewHealthIndicator(test.brokers)
			result := indicator.Health(t.Context())
			assert.True(t, result.Healthy)
			assert.True(t, result.Degraded)
			assert.NotEmpty(t, result.Reason)
			assert.True(t, indicator.Readiness(t.Context()).Ready)
		})
	}
}

func TestKafkaHealthIndicator_CancelledPollDoesNotPoisonCache(t *testing.T) {
	t.Parallel()

	broker := sarama.NewMockBroker(t, 1)
	t.Cleanup(broker.Close)
	broker.SetHandlerByMap(map[string]sarama.MockResponse{
		"MetadataRequest": sarama.NewMockMetadataResponse(t).SetBroker(broker.Addr(), broker.BrokerID()),
	})
	indicator := NewHealthIndicator([]string{broker.Addr()})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	assert.True(t, indicator.Health(ctx).Degraded)
	assert.Empty(t, broker.History())
	assert.False(t, indicator.Health(t.Context()).Degraded)
}

func TestKafkaHealthIndicator_MetadataTimeout(t *testing.T) {
	t.Parallel()

	broker := sarama.NewMockBroker(t, 1)
	t.Cleanup(broker.Close)
	indicator := NewHealthIndicator([]string{broker.Addr()})
	started := time.Now()

	assert.True(t, indicator.Health(t.Context()).Degraded)
	assert.Less(t, time.Since(started), 2*time.Second, "a silent broker must not hang health polls")
}
