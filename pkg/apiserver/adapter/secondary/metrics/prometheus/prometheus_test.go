package prometheus_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/secondary/metrics/prometheus"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
)

// vectorResponse builds a Prometheus HTTP API instant-vector response body with
// the given scalar sample values.
func vectorResponse(values ...string) string {
	results := make([]string, 0, len(values))
	for i, v := range values {
		results = append(results, `{"metric":{"exporter":"e`+string(rune('0'+i))+`"},"value":[1700000000,"`+v+`"]}`)
	}

	return `{"status":"success","data":{"resultType":"vector","result":[` +
		strings.Join(results, ",") + `]}}`
}

func newFakeBackend(t *testing.T, body string, capture *url.Values) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		assert.NoError(t, r.ParseForm())

		if capture != nil {
			*capture = r.Form
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
}

func newEndpoint(query *agentmodel.EndpointMetricsQuery) *agentmodel.Endpoint {
	return &agentmodel.Endpoint{
		Metadata: agentmodel.EndpointMetadata{
			Name:       "victoria",
			Namespace:  "monitoring",
			Attributes: agentmodel.Attributes{"exporter": "prometheusremotewrite"},
			CreatedAt:  time.Now(),
			DeletedAt:  nil,
		},
		Spec: agentmodel.EndpointSpec{
			URL:          "http://vm/insert",
			Protocol:     "prometheusremotewrite",
			Signals:      agentmodel.EndpointSignals{Metrics: true, Logs: false, Traces: false},
			Tenants:      nil,
			MetricsQuery: query,
		},
		Status: agentmodel.EndpointStatus{Conditions: nil},
	}
}

func TestQueryEndpointThroughput_SumsVectorPerSignal(t *testing.T) {
	t.Parallel()

	server := newFakeBackend(t, vectorResponse("1.5", "2.5"), nil)
	defer server.Close()

	adapter, err := prometheus.NewAdapter(server.URL, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	endpoint := newEndpoint(&agentmodel.EndpointMetricsQuery{
		Metrics: `sum(rate(otelcol_exporter_sent_metric_points_total[{{.Window}}]))`,
		Logs:    "",
		Traces:  "",
	})

	got, err := adapter.QueryEndpointThroughput(context.Background(), endpoint, 5*time.Minute, time.Unix(1700000000, 0))
	require.NoError(t, err)

	assert.True(t, got.Metrics.Measured)
	assert.InDelta(t, 4.0, got.Metrics.PerSecond, 1e-9)
	assert.Equal(t, 2, got.Metrics.SeriesCount)

	// Signals without a template stay unmeasured.
	assert.False(t, got.Logs.Measured)
	assert.False(t, got.Traces.Measured)
	assert.Equal(t, "monitoring", got.Namespace)
	assert.Equal(t, 5*time.Minute, got.Window)
}

func TestQueryEndpointThroughput_RendersWindowAndIdentity(t *testing.T) {
	t.Parallel()

	var captured url.Values

	server := newFakeBackend(t, vectorResponse("1"), &captured)
	defer server.Close()

	adapter, err := prometheus.NewAdapter(server.URL, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	endpoint := newEndpoint(&agentmodel.EndpointMetricsQuery{
		Metrics: `sum(rate(otelcol_exporter_sent_metric_points_total{exporter="{{.Attributes.exporter}}"}[{{.Window}}]))`,
		Logs:    "",
		Traces:  "",
	})

	_, err = adapter.QueryEndpointThroughput(context.Background(), endpoint, 90*time.Second, time.Unix(1700000000, 0))
	require.NoError(t, err)

	rendered := captured.Get("query")
	assert.Contains(t, rendered, `exporter="prometheusremotewrite"`)
	assert.Contains(t, rendered, "[90s]")
}

func TestQueryEndpointThroughput_NilQuerySkipsBackend(t *testing.T) {
	t.Parallel()

	called := false

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	}))
	defer server.Close()

	adapter, err := prometheus.NewAdapter(server.URL, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	got, err := adapter.QueryEndpointThroughput(
		context.Background(), newEndpoint(nil), time.Minute, time.Unix(1700000000, 0))
	require.NoError(t, err)

	assert.False(t, called, "backend must not be queried when no template is configured")
	assert.False(t, got.Metrics.Measured)
}
