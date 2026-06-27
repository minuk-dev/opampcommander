package agentservice_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/persistence/inmemory"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentservice "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/service"
)

// fakeMetricsPort records the arguments it was called with and returns a fixed
// measured throughput so the service's aggregation/passthrough can be asserted.
type fakeMetricsPort struct {
	calls      int
	lastWindow time.Duration
	lastAt     time.Time
}

func (f *fakeMetricsPort) QueryEndpointThroughput(
	_ context.Context,
	endpoint *agentmodel.Endpoint,
	window time.Duration,
	evaluatedAt time.Time,
) (*agentmodel.EndpointThroughput, error) {
	f.calls++
	f.lastWindow = window
	f.lastAt = evaluatedAt

	return &agentmodel.EndpointThroughput{
		Namespace:   endpoint.Metadata.Namespace,
		Name:        endpoint.Metadata.Name,
		EvaluatedAt: evaluatedAt,
		Window:      window,
		Metrics:     agentmodel.SignalThroughput{Measured: true, PerSecond: 1, SeriesCount: 1},
		Logs:        agentmodel.SignalThroughput{Measured: false, PerSecond: 0, SeriesCount: 0},
		Traces:      agentmodel.SignalThroughput{Measured: false, PerSecond: 0, SeriesCount: 0},
	}, nil
}

func putEndpoint(t *testing.T, repo *inmemory.EndpointRepository, namespace, name string) {
	t.Helper()

	endpoint := agentmodel.NewEndpoint(namespace, name, nil, time.Now(), "test")
	_, err := repo.PutEndpoint(context.Background(), endpoint)
	require.NoError(t, err)
}

func TestEndpointMetricsService_GetEndpointThroughput(t *testing.T) {
	t.Parallel()

	repo := inmemory.NewEndpointRepository()
	putEndpoint(t, repo, "monitoring", "vm")

	metrics := &fakeMetricsPort{}
	service := agentservice.NewEndpointMetricsService(repo, metrics)

	at := time.Unix(1700000000, 0)

	got, err := service.GetEndpointThroughput(context.Background(), "monitoring", "vm", 5*time.Minute, at)
	require.NoError(t, err)

	assert.Equal(t, "vm", got.Name)
	assert.Equal(t, "monitoring", got.Namespace)
	assert.True(t, got.Metrics.Measured)
	// The window and instant are passed through to the metrics port unchanged.
	assert.Equal(t, 5*time.Minute, metrics.lastWindow)
	assert.Equal(t, at, metrics.lastAt)
}

func TestEndpointMetricsService_GetEndpointThroughput_NotFound(t *testing.T) {
	t.Parallel()

	repo := inmemory.NewEndpointRepository()
	service := agentservice.NewEndpointMetricsService(repo, &fakeMetricsPort{})

	_, err := service.GetEndpointThroughput(
		context.Background(), "monitoring", "missing", time.Minute, time.Now())
	require.Error(t, err)
}

func TestEndpointMetricsService_ListEndpointThroughput(t *testing.T) {
	t.Parallel()

	repo := inmemory.NewEndpointRepository()
	putEndpoint(t, repo, "monitoring", "vm")
	putEndpoint(t, repo, "monitoring", "loki")
	// An endpoint in another namespace must not leak into the result.
	putEndpoint(t, repo, "other", "tempo")

	metrics := &fakeMetricsPort{}
	service := agentservice.NewEndpointMetricsService(repo, metrics)

	got, err := service.ListEndpointThroughput(context.Background(), "monitoring", time.Minute, time.Now())
	require.NoError(t, err)

	assert.Len(t, got, 2)
	assert.Equal(t, 2, metrics.calls)

	names := []string{got[0].Name, got[1].Name}
	assert.ElementsMatch(t, []string{"vm", "loki"}, names)
}
