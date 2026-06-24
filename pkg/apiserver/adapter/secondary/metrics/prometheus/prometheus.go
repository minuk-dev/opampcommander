// Package prometheus provides an outbound adapter that implements
// [agentport.EndpointMetricsQueryPort] against a Prometheus-compatible HTTP API
// (Prometheus, VictoriaMetrics, Mimir, ...). It evaluates an endpoint's
// per-signal PromQL templates to measure how much telemetry collectors are
// sending to that endpoint.
package prometheus

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"text/template"
	"time"

	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
)

var _ agentport.EndpointMetricsQueryPort = (*Adapter)(nil)

// Adapter queries a Prometheus-compatible backend for endpoint throughput.
type Adapter struct {
	api    promv1.API
	logger *slog.Logger
}

// NewAdapter creates an Adapter targeting the Prometheus-compatible HTTP API at
// address (e.g. "http://prometheus:9090" or a VictoriaMetrics vmselect URL).
func NewAdapter(address string, logger *slog.Logger) (*Adapter, error) {
	//exhaustruct:ignore // only Address is needed; Client/RoundTripper default.
	client, err := promapi.NewClient(promapi.Config{Address: address})
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus client for %q: %w", address, err)
	}

	return newAdapterWithAPI(promv1.NewAPI(client), logger), nil
}

// newAdapterWithAPI builds an Adapter from an already-constructed API. It exists
// so tests can inject a client pointed at an httptest server.
func newAdapterWithAPI(api promv1.API, logger *slog.Logger) *Adapter {
	return &Adapter{api: api, logger: logger}
}

// queryTemplateData is the context exposed to an endpoint's PromQL templates.
type queryTemplateData struct {
	// Namespace and Name are the endpoint identity.
	Namespace string
	Name      string
	// URL and Protocol come from the endpoint spec.
	URL      string
	Protocol string
	// Attributes are the endpoint's user-supplied attributes.
	Attributes map[string]string
	// Window is the rate window as a PromQL duration literal (e.g. "300s"),
	// suitable for direct use inside a range selector like rate(...[{{.Window}}]).
	Window string
}

// QueryEndpointThroughput implements [agentport.EndpointMetricsQueryPort].
func (a *Adapter) QueryEndpointThroughput(
	ctx context.Context,
	endpoint *agentmodel.Endpoint,
	window time.Duration,
	evaluatedAt time.Time,
) (*agentmodel.EndpointThroughput, error) {
	result := &agentmodel.EndpointThroughput{
		Namespace:   endpoint.Metadata.Namespace,
		Name:        endpoint.Metadata.Name,
		EvaluatedAt: evaluatedAt,
		Window:      window,
		Metrics:     agentmodel.SignalThroughput{Measured: false, PerSecond: 0, SeriesCount: 0},
		Logs:        agentmodel.SignalThroughput{Measured: false, PerSecond: 0, SeriesCount: 0},
		Traces:      agentmodel.SignalThroughput{Measured: false, PerSecond: 0, SeriesCount: 0},
	}

	query := endpoint.Spec.MetricsQuery
	if query.IsZero() {
		return result, nil
	}

	data := newQueryTemplateData(endpoint, window)

	var err error

	result.Metrics, err = a.querySignal(ctx, "metrics", query.Metrics, data, evaluatedAt)
	if err != nil {
		return nil, err
	}

	result.Logs, err = a.querySignal(ctx, "logs", query.Logs, data, evaluatedAt)
	if err != nil {
		return nil, err
	}

	result.Traces, err = a.querySignal(ctx, "traces", query.Traces, data, evaluatedAt)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func newQueryTemplateData(endpoint *agentmodel.Endpoint, window time.Duration) queryTemplateData {
	return queryTemplateData{
		Namespace:  endpoint.Metadata.Namespace,
		Name:       endpoint.Metadata.Name,
		URL:        endpoint.Spec.URL,
		Protocol:   endpoint.Spec.Protocol,
		Attributes: endpoint.Metadata.Attributes,
		Window:     promQLDuration(window),
	}
}

func (a *Adapter) querySignal(
	ctx context.Context,
	signal string,
	tmpl string,
	data queryTemplateData,
	evaluatedAt time.Time,
) (agentmodel.SignalThroughput, error) {
	unmeasured := agentmodel.SignalThroughput{Measured: false, PerSecond: 0, SeriesCount: 0}
	if strings.TrimSpace(tmpl) == "" {
		return unmeasured, nil
	}

	query, err := renderQuery(signal, tmpl, data)
	if err != nil {
		return unmeasured, err
	}

	value, warnings, err := a.api.Query(ctx, query, evaluatedAt)
	if err != nil {
		return unmeasured, fmt.Errorf("failed to query %s throughput: %w", signal, err)
	}

	for _, warning := range warnings {
		a.logger.Warn("prometheus query warning",
			slog.String("signal", signal),
			slog.String("warning", warning),
		)
	}

	sum, count := sumValue(value)

	return agentmodel.SignalThroughput{Measured: true, PerSecond: sum, SeriesCount: count}, nil
}

func renderQuery(signal, tmpl string, data queryTemplateData) (string, error) {
	parsed, err := template.New(signal).Option("missingkey=error").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse %s query template: %w", signal, err)
	}

	var buf strings.Builder

	err = parsed.Execute(&buf, data)
	if err != nil {
		return "", fmt.Errorf("failed to render %s query template: %w", signal, err)
	}

	return buf.String(), nil
}

// sumValue aggregates a PromQL result into a single total and the number of
// contributing series. It sums instant-vector samples (the expected shape: a
// per-series rate to be totaled) and accepts a bare scalar; NaN samples (e.g. a
// rate over a gap) are skipped and not counted. Any other value type yields zero.
func sumValue(value model.Value) (float64, int) {
	switch typed := value.(type) {
	case model.Vector:
		var (
			sum   float64
			count int
		)

		for _, sample := range typed {
			if sample == nil || math.IsNaN(float64(sample.Value)) {
				continue
			}

			sum += float64(sample.Value)
			count++
		}

		return sum, count
	case *model.Scalar:
		if math.IsNaN(float64(typed.Value)) {
			return 0, 0
		}

		return float64(typed.Value), 1
	default:
		return 0, 0
	}
}

// promQLDuration renders a duration as a PromQL duration literal. It uses whole
// seconds (e.g. "300s"), which PromQL always accepts inside a range selector and
// avoids the "5m0s" form produced by time.Duration.String that PromQL rejects.
func promQLDuration(d time.Duration) string {
	seconds := max(int64(d.Seconds()), 1)

	return fmt.Sprintf("%ds", seconds)
}
