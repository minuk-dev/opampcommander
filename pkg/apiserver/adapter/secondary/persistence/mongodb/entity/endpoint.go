package entity

import (
	"time"

	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// EndpointNameFieldName is the field name for name in MongoDB.
const EndpointNameFieldName = "metadata.name"

// EndpointResourceEntity is the MongoDB entity for endpoint resource.
type EndpointResourceEntity struct {
	ID       *bson.ObjectID               `bson:"_id,omitempty"`
	Metadata EndpointResourceMetadata     `bson:"metadata"`
	Spec     EndpointResourceSpec         `bson:"spec"`
	Status   EndpointResourceEntityStatus `bson:"status"`
}

// EndpointResourceMetadata represents the metadata of an endpoint resource.
type EndpointResourceMetadata struct {
	Name       string            `bson:"name"`
	Namespace  string            `bson:"namespace"`
	Attributes map[string]string `bson:"attributes,omitempty"`
	CreatedAt  time.Time         `bson:"createdAt"`
	DeletedAt  *time.Time        `bson:"deletedAt,omitempty"`
}

// EndpointResourceSpec represents the specification of an endpoint resource.
type EndpointResourceSpec struct {
	URL          string                        `bson:"url"`
	Protocol     string                        `bson:"protocol"`
	Signals      EndpointResourceSignals       `bson:"signals"`
	Tenants      []EndpointResourceTenant      `bson:"tenants,omitempty"`
	MetricsQuery *EndpointResourceMetricsQuery `bson:"metricsQuery,omitempty"`
}

// EndpointResourceMetricsQuery represents the per-signal PromQL templates used to
// measure endpoint throughput.
type EndpointResourceMetricsQuery struct {
	Metrics string `bson:"metrics,omitempty"`
	Logs    string `bson:"logs,omitempty"`
	Traces  string `bson:"traces,omitempty"`
}

// EndpointResourceSignals represents the supported signals of an endpoint resource.
type EndpointResourceSignals struct {
	Metrics bool `bson:"metrics"`
	Logs    bool `bson:"logs"`
	Traces  bool `bson:"traces"`
}

// EndpointResourceTenant represents a tenant of an endpoint resource.
type EndpointResourceTenant struct {
	Name    string                   `bson:"name"`
	Headers map[string]string        `bson:"headers,omitempty"`
	Tags    map[string]string        `bson:"tags,omitempty"`
	Signals *EndpointResourceSignals `bson:"signals,omitempty"`
}

// EndpointResourceEntityStatus represents the status of an endpoint resource.
type EndpointResourceEntityStatus struct {
	Conditions []Condition `bson:"conditions,omitempty"`
}

// ToDomain converts the entity to domain model.
func (e *EndpointResourceEntity) ToDomain() *agentmodel.Endpoint {
	return &agentmodel.Endpoint{
		Metadata: agentmodel.EndpointMetadata{
			Name:       e.Metadata.Name,
			Namespace:  e.Metadata.Namespace,
			Attributes: e.Metadata.Attributes,
			CreatedAt:  e.Metadata.CreatedAt,
			DeletedAt:  e.Metadata.DeletedAt,
		},
		Spec: agentmodel.EndpointSpec{
			URL:      e.Spec.URL,
			Protocol: e.Spec.Protocol,
			Signals:  e.Spec.Signals.toDomain(),
			Tenants: lo.Map(e.Spec.Tenants, func(t EndpointResourceTenant, _ int) agentmodel.EndpointTenant {
				return t.toDomain()
			}),
			MetricsQuery: e.Spec.MetricsQuery.toDomain(),
		},
		Status: agentmodel.EndpointStatus{
			Conditions: lo.Map(e.Status.Conditions, func(c Condition, _ int) model.Condition {
				return c.ToDomain()
			}),
		},
	}
}

func (q *EndpointResourceMetricsQuery) toDomain() *agentmodel.EndpointMetricsQuery {
	if q == nil {
		return nil
	}

	return &agentmodel.EndpointMetricsQuery{
		Metrics: q.Metrics,
		Logs:    q.Logs,
		Traces:  q.Traces,
	}
}

func (s EndpointResourceSignals) toDomain() agentmodel.EndpointSignals {
	return agentmodel.EndpointSignals{
		Metrics: s.Metrics,
		Logs:    s.Logs,
		Traces:  s.Traces,
	}
}

func (t EndpointResourceTenant) toDomain() agentmodel.EndpointTenant {
	var signals *agentmodel.EndpointSignals

	if t.Signals != nil {
		s := t.Signals.toDomain()
		signals = &s
	}

	return agentmodel.EndpointTenant{
		Name:    t.Name,
		Headers: t.Headers,
		Tags:    t.Tags,
		Signals: signals,
	}
}

// EndpointResourceEntityFromDomain converts domain model to entity.
func EndpointResourceEntityFromDomain(
	endpoint *agentmodel.Endpoint,
) *EndpointResourceEntity {
	//nolint:exhaustruct // ID is set by MongoDB
	return &EndpointResourceEntity{
		Metadata: EndpointResourceMetadata{
			Name:       endpoint.Metadata.Name,
			Namespace:  endpoint.Metadata.Namespace,
			Attributes: endpoint.Metadata.Attributes,
			CreatedAt:  endpoint.Metadata.CreatedAt,
			DeletedAt:  endpoint.Metadata.DeletedAt,
		},
		Spec: EndpointResourceSpec{
			URL:      endpoint.Spec.URL,
			Protocol: endpoint.Spec.Protocol,
			Signals:  endpointResourceSignalsFromDomain(endpoint.Spec.Signals),
			Tenants: lo.Map(endpoint.Spec.Tenants, func(t agentmodel.EndpointTenant, _ int) EndpointResourceTenant {
				return endpointResourceTenantFromDomain(t)
			}),
			MetricsQuery: endpointResourceMetricsQueryFromDomain(endpoint.Spec.MetricsQuery),
		},
		Status: EndpointResourceEntityStatus{
			Conditions: lo.Map(endpoint.Status.Conditions, func(c model.Condition, _ int) Condition {
				return NewConditionFromDomain(c)
			}),
		},
	}
}

func endpointResourceMetricsQueryFromDomain(
	query *agentmodel.EndpointMetricsQuery,
) *EndpointResourceMetricsQuery {
	if query == nil {
		return nil
	}

	return &EndpointResourceMetricsQuery{
		Metrics: query.Metrics,
		Logs:    query.Logs,
		Traces:  query.Traces,
	}
}

func endpointResourceSignalsFromDomain(s agentmodel.EndpointSignals) EndpointResourceSignals {
	return EndpointResourceSignals{
		Metrics: s.Metrics,
		Logs:    s.Logs,
		Traces:  s.Traces,
	}
}

func endpointResourceTenantFromDomain(tenant agentmodel.EndpointTenant) EndpointResourceTenant {
	var signals *EndpointResourceSignals

	if tenant.Signals != nil {
		s := endpointResourceSignalsFromDomain(*tenant.Signals)
		signals = &s
	}

	return EndpointResourceTenant{
		Name:    tenant.Name,
		Headers: tenant.Headers,
		Tags:    tenant.Tags,
		Signals: signals,
	}
}
