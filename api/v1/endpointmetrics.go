package v1

const (
	// EndpointThroughputKind is the kind for endpoint throughput results.
	EndpointThroughputKind = "EndpointThroughput"
)

// EndpointThroughput is how much telemetry collectors are currently sending to an
// endpoint, broken down by signal. Each per-signal value is a per-second rate
// summed across the contributing collector time series, evaluated over Window at
// EvaluatedAt. Units differ per signal (metric data points, log records, spans —
// all per second) and are not normalized.
type EndpointThroughput struct {
	// Namespace and Name identify the endpoint the throughput was measured for.
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	// EvaluatedAt is the instant the rates were evaluated at.
	EvaluatedAt Time `json:"evaluatedAt"`
	// Window is the rate window the per-second values were computed over,
	// formatted as a Go duration (e.g. "5m0s").
	Window string `json:"window"`
	// Metrics is the metric-data-point send rate (points/sec).
	Metrics SignalThroughput `json:"metrics"`
	// Logs is the log-record send rate (records/sec).
	Logs SignalThroughput `json:"logs"`
	// Traces is the span send rate (spans/sec).
	Traces SignalThroughput `json:"traces"`
} // @name EndpointThroughput

// SignalThroughput is the aggregated send rate for one telemetry signal.
type SignalThroughput struct {
	// Measured reports whether a query was configured and executed for the signal.
	// When false, PerSecond and SeriesCount are zero and meaningless: the signal is
	// simply not tracked for the endpoint.
	Measured bool `json:"measured"`
	// PerSecond is the aggregated per-second rate summed across contributing series.
	PerSecond float64 `json:"perSecond"`
	// SeriesCount is the number of contributing time series the rate was summed over.
	SeriesCount int `json:"seriesCount"`
} // @name SignalThroughput
