package v1

const (
	// EndpointKind is the kind for Endpoint resources.
	EndpointKind = "Endpoint"
)

// Endpoint represents a telemetry backend/destination resource.
type Endpoint struct {
	Kind       string           `json:"kind"`
	APIVersion string           `json:"apiVersion"`
	Metadata   EndpointMetadata `json:"metadata"`
	Spec       EndpointSpec     `json:"spec"`
	Status     EndpointStatus   `json:"status"`
} // @name Endpoint

// EndpointMetadata represents the metadata of an endpoint.
type EndpointMetadata struct {
	Name       string     `json:"name"`
	Namespace  string     `json:"namespace"`
	Attributes Attributes `json:"attributes"`
	CreatedAt  Time       `json:"createdAt"`
} // @name EndpointMetadata

// EndpointSpec represents the specification of an endpoint.
type EndpointSpec struct {
	// URL is the destination of the telemetry backend.
	URL string `json:"url"`
	// Protocol is the export protocol (e.g. "otlp", "otlphttp", "prometheusremotewrite").
	Protocol string `json:"protocol"`
	// Signals declares which telemetry signals this endpoint supports by default.
	Signals EndpointSignals `json:"signals"`
	// Tenants is the multi-tenant breakdown of the endpoint.
	Tenants []EndpointTenant `json:"tenants,omitempty"`
} // @name EndpointSpec

// EndpointSignals declares which telemetry signals are supported.
type EndpointSignals struct {
	Metrics bool `json:"metrics"`
	Logs    bool `json:"logs"`
	Traces  bool `json:"traces"`
} // @name EndpointSignals

// EndpointTenant represents a tenant of an endpoint.
type EndpointTenant struct {
	// Name is the tenant identifier, unique within the endpoint.
	Name string `json:"name"`
	// Headers are per-tenant headers (e.g. "X-Scope-OrgID" for Mimir/Loki/Tempo).
	Headers map[string]string `json:"headers,omitempty"`
	// Tags are arbitrary key/value labels for internal differentiation/selection.
	Tags map[string]string `json:"tags,omitempty"`
	// Signals optionally overrides the endpoint-level signal support for this tenant.
	Signals *EndpointSignals `json:"signals,omitempty"`
} // @name EndpointTenant

// EndpointStatus represents the status of an endpoint.
type EndpointStatus struct {
	Conditions []Condition `json:"conditions"`
} // @name EndpointStatus
