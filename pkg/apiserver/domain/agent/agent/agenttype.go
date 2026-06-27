package agent

// Type is an extensible label for the agent's kind and, for OpenTelemetry
// Collectors, its distribution. It is derived from the reported "service.name"
// identifying attribute.
//
// Like Platform it is an open enum: the well-known OpenTelemetry Collector
// distributions have named constants for documentation and comparison, but any
// "otelcol"/"otelcol-<distro>" service.name is preserved verbatim so a new
// distribution surfaces as its own value without a code change. Anything that
// does not follow the Collector naming convention is TypeUnknown.
type Type string

const (
	// TypeOTelCollector is the upstream OpenTelemetry Collector core
	// distribution, which reports service.name "otelcol".
	TypeOTelCollector Type = "otelcol"
	// TypeOTelCollectorContrib is the contrib distribution, which reports
	// service.name "otelcol-contrib".
	TypeOTelCollectorContrib Type = "otelcol-contrib"
	// TypeOTelCollectorK8s is the Kubernetes distribution, which reports
	// service.name "otelcol-k8s".
	TypeOTelCollectorK8s Type = "otelcol-k8s"
	// TypeUnknown means the agent kind could not be determined from
	// service.name.
	TypeUnknown Type = "unknown"
)

// otelCollectorPrefix is the naming convention shared by every official
// OpenTelemetry Collector distribution: the core binary is "otelcol" and each
// distribution appends a "-<distro>" suffix (e.g. "otelcol-contrib").
const otelCollectorPrefix = "otelcol"

// IsOTelCollector reports whether the type denotes an OpenTelemetry Collector of
// any distribution.
func (t Type) IsOTelCollector() bool {
	return t != TypeUnknown
}
