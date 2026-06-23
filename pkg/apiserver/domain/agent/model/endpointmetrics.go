package agentmodel

import "time"

// EndpointThroughput is the rate at which collectors are sending telemetry to a
// single endpoint, broken down by signal. Each per-signal value is an aggregate
// (sum) across every contributing collector/exporter time series, expressed as a
// per-second rate evaluated over Window at instant EvaluatedAt.
//
// It is the result of evaluating an endpoint's EndpointMetricsQuery against the
// metrics backend; the units differ per signal (metric data points, log records,
// spans — all per second) and are intentionally not normalized.
type EndpointThroughput struct {
	// Namespace and Name identify the endpoint the throughput was measured for.
	Namespace string
	Name      string
	// EvaluatedAt is the instant the rates were evaluated at.
	EvaluatedAt time.Time
	// Window is the rate window the per-second values were computed over.
	Window time.Duration
	// Metrics is the metric-data-point send rate (points/sec).
	Metrics SignalThroughput
	// Logs is the log-record send rate (records/sec).
	Logs SignalThroughput
	// Traces is the span send rate (spans/sec).
	Traces SignalThroughput
}

// SignalThroughput is the aggregated send rate for one telemetry signal.
type SignalThroughput struct {
	// Measured reports whether a query was configured and executed for this
	// signal. When false, PerSecond and SeriesCount are zero and meaningless: the
	// signal is simply not tracked for the endpoint.
	Measured bool
	// PerSecond is the aggregated per-second rate summed across contributing
	// time series.
	PerSecond float64
	// SeriesCount is the number of contributing time series (e.g. distinct
	// collectors/exporters) that the rate was summed over.
	SeriesCount int
}
