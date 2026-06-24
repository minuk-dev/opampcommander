package config

import "time"

// MetricsBackendSettings configures the metrics backend that endpoint-throughput
// queries run against. When Type is empty or "none", the no-op adapter is wired
// and every endpoint's throughput is reported as unmeasured.
type MetricsBackendSettings struct {
	// Type selects the backend implementation. "prometheus" enables PromQL
	// queries against a Prometheus-compatible HTTP API; "" / "none" disables them.
	Type MetricsBackendType
	// Address is the base URL of the Prometheus-compatible HTTP API
	// (e.g. "http://prometheus:9090" or a VictoriaMetrics vmselect endpoint).
	Address string
	// DefaultWindow is the rate window used when a caller does not specify one.
	DefaultWindow time.Duration
}

// MetricsBackendType is the type of metrics backend used for throughput queries.
type MetricsBackendType string

const (
	// MetricsBackendTypePrometheus queries a Prometheus-compatible HTTP API.
	MetricsBackendTypePrometheus MetricsBackendType = "prometheus"
	// MetricsBackendTypeNone disables throughput queries (no-op adapter).
	MetricsBackendTypeNone MetricsBackendType = "none"
)
