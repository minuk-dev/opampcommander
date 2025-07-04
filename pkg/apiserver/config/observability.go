package config

import "log/slog"

// ObservabilitySettings holds the settings for observability features.
type ObservabilitySettings struct {
	// ServiceName is the name of the service for which observability is configured.
	// It used for naming traces, metrics, and logs.
	ServiceName string

	Metric MetricSettings
	Log    LogSettings
	Trace  TraceSettings
}

// MetricSettings holds the settings for metrics collection.
type MetricSettings struct {
	// Enabled indicates whether metrics collection is enabled.
	Enabled bool

	// Type specifies the type of metrics to be used.
	Type MetricType

	// Endpoint specifies the endpoint for metrics collection.
	// When using Prometheus, this is the endpoint to expose metrics.
	// When using OpenTelemetry, this is the endpoint to send metrics to.
	// It should be a valid URL or URI.
	Endpoint string
}

// MetricType represents the type of metrics to be used.
type MetricType string

const (
	// MetricTypePrometheus represents Prometheus metrics.
	MetricTypePrometheus MetricType = "prometheus"
	// MetricTypeOTel represents OpenTelemetry metrics.
	MetricTypeOTel MetricType = "otel"
)

// LogSettings holds the settings for logging.
// It only supports stdout/stderr logging for now.
type LogSettings struct {
	// Enabled indicates whether logging is enabled.
	Enabled bool
	// Level specifies the log level.
	Level slog.Level
	// Format specifies the log format (e.g., text or json).
	Format LogFormat
}

// LogFormat is a string type that represents the log format.
type LogFormat string

const (
	// LogFormatText represents the text log format.
	LogFormatText LogFormat = "text"
	// LogFormatJSON represents the JSON log format.
	LogFormatJSON LogFormat = "json"
)

// TraceSettings holds the settings for tracing.
type TraceSettings struct {
	// Enabled indicates whether tracing is enabled.
	Enabled bool
	// Endpoint specifies the endpoint for tracing.
	Endpoint string
}
