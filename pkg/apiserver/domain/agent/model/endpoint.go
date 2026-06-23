package agentmodel

import (
	"time"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// Endpoint is a domain aggregate that represents a telemetry backend/destination
// that managed OpenTelemetry collectors/agents export to (e.g. an OTLP backend,
// Tempo, Loki, or Mimir).
//
// An Endpoint declares which telemetry signals (metrics/logs/traces) it supports
// and is multi-tenant aware: the same destination can be managed differently per
// tenant via per-tenant Headers (e.g. the "X-Scope-OrgID" header used by
// Mimir/Loki/Tempo) and Tags. It is a referenceable resource — remote configs
// reference it by namespace+name(+tenant); this type does not itself materialize
// collector exporter configuration.
type Endpoint struct {
	Metadata EndpointMetadata
	Spec     EndpointSpec
	Status   EndpointStatus
}

// EndpointMetadata contains identity and lifecycle information for an endpoint.
// Together, Namespace and Name form the unique identity of the endpoint.
type EndpointMetadata struct {
	// Name is the name of the endpoint.
	Name string
	// Namespace is the namespace of the endpoint.
	Namespace string
	// Attributes is a map of user-supplied attributes for the endpoint.
	Attributes Attributes
	// CreatedAt is the timestamp when the endpoint was created.
	CreatedAt time.Time
	// DeletedAt is the timestamp when the endpoint was soft deleted.
	// If nil, the endpoint is not deleted.
	DeletedAt *time.Time
}

// EndpointSpec contains the specification for the endpoint.
type EndpointSpec struct {
	// URL is the destination of the telemetry backend.
	URL string
	// Protocol is the export protocol (e.g. "otlp", "otlphttp",
	// "prometheusremotewrite"). It is a free-form string.
	Protocol string
	// Signals declares which telemetry signals this endpoint supports by default.
	Signals EndpointSignals
	// Tenants is the multi-tenant breakdown. The same endpoint can be managed
	// differently per tenant via per-tenant Headers and Tags.
	Tenants []EndpointTenant
	// MetricsQuery configures how to measure, from a metrics backend, how much
	// telemetry collectors are sending to this endpoint. Nil means throughput is
	// not tracked for this endpoint.
	MetricsQuery *EndpointMetricsQuery
}

// EndpointMetricsQuery holds a PromQL query template per telemetry signal used
// to measure how much data collectors are sending to this endpoint.
//
// Each template is rendered (text/template) with a small context — notably the
// rate window and the endpoint's own identity — before being executed against
// the metrics backend. An empty template means that signal is not measured. A
// rendered query is expected to evaluate to an instant vector of per-second
// rates (e.g. sum by (...) (rate(otelcol_exporter_sent_metric_points_total[...])));
// the adapter sums the result series to get the endpoint's total throughput for
// that signal.
type EndpointMetricsQuery struct {
	// Metrics is the PromQL template measuring metric data points per second.
	Metrics string
	// Logs is the PromQL template measuring log records per second.
	Logs string
	// Traces is the PromQL template measuring spans per second.
	Traces string
}

// IsZero reports whether no per-signal template is configured.
func (q *EndpointMetricsQuery) IsZero() bool {
	return q == nil || (q.Metrics == "" && q.Logs == "" && q.Traces == "")
}

// EndpointSignals declares which telemetry signals are supported.
type EndpointSignals struct {
	// Metrics reports whether the metrics signal is supported.
	Metrics bool
	// Logs reports whether the logs signal is supported.
	Logs bool
	// Traces reports whether the traces signal is supported.
	Traces bool
}

// EndpointTenant represents a tenant of an endpoint. It lets the same physical
// destination be managed differently per tenant.
type EndpointTenant struct {
	// Name is the tenant identifier, unique within the endpoint.
	Name string
	// Headers are per-tenant headers (e.g. "X-Scope-OrgID" for Mimir/Loki/Tempo).
	Headers map[string]string
	// Tags are arbitrary key/value labels for internal differentiation/selection.
	Tags map[string]string
	// Signals optionally overrides the endpoint-level signal support for this
	// tenant. If nil, the tenant inherits the endpoint's Signals.
	Signals *EndpointSignals
}

// EndpointStatus contains the observed state of the endpoint.
type EndpointStatus struct {
	// Conditions is a list of conditions that apply to the endpoint.
	Conditions []model.Condition
}

// NewEndpoint creates a new endpoint with the given identity, marking it as
// created by createdBy at createdAt.
func NewEndpoint(
	namespace string,
	name string,
	attributes Attributes,
	createdAt time.Time,
	createdBy string,
) *Endpoint {
	return &Endpoint{
		Metadata: EndpointMetadata{
			Name:       name,
			Namespace:  namespace,
			Attributes: attributes,
			CreatedAt:  createdAt,
			DeletedAt:  nil,
		},
		Spec: EndpointSpec{
			URL:          "",
			Protocol:     "",
			Signals:      EndpointSignals{Metrics: false, Logs: false, Traces: false},
			Tenants:      nil,
			MetricsQuery: nil,
		},
		Status: EndpointStatus{
			Conditions: []model.Condition{
				{
					Type:               model.ConditionTypeCreated,
					Status:             model.ConditionStatusTrue,
					LastTransitionTime: createdAt,
					Reason:             createdBy,
					Message:            "Endpoint created",
				},
			},
		},
	}
}

// IsDeleted returns true if the endpoint is marked as deleted.
func (e *Endpoint) IsDeleted() bool {
	return e.Metadata.DeletedAt != nil
}

// MarkDeleted marks the endpoint as deleted by setting DeletedAt and adding a
// deleted condition.
func (e *Endpoint) MarkDeleted(deletedAt time.Time, deletedBy string) {
	e.Metadata.DeletedAt = &deletedAt
	e.Status.Conditions = append(e.Status.Conditions, model.Condition{
		Type:               model.ConditionTypeDeleted,
		Status:             model.ConditionStatusTrue,
		LastTransitionTime: deletedAt,
		Reason:             deletedBy,
		Message:            "Endpoint deleted",
	})
}

// Tenant returns the tenant with the given name, or nil if no such tenant exists.
func (e *Endpoint) Tenant(name string) *EndpointTenant {
	for i := range e.Spec.Tenants {
		if e.Spec.Tenants[i].Name == name {
			return &e.Spec.Tenants[i]
		}
	}

	return nil
}

// EffectiveSignals returns the signals that apply to the given tenant: the
// tenant's own override when present, otherwise the endpoint-level signals.
// An empty tenant name (or an unknown tenant) yields the endpoint-level signals.
func (e *Endpoint) EffectiveSignals(tenant string) EndpointSignals {
	if tenant != "" {
		if t := e.Tenant(tenant); t != nil && t.Signals != nil {
			return *t.Signals
		}
	}

	return e.Spec.Signals
}
