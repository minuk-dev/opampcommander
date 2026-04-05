package entity

import (
	"time"

	"github.com/samber/lo"

	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

const (
	// NamespaceKeyFieldName is the key field name for namespace.
	NamespaceKeyFieldName = "metadata.name"
)

// Namespace is the MongoDB entity for namespace.
type Namespace struct {
	Common `bson:",inline"`

	Metadata NamespaceMetadata       `bson:"metadata"`
	Status   NamespaceResourceStatus `bson:"status"`
}

// NamespaceMetadata represents the metadata of a namespace.
type NamespaceMetadata struct {
	Name        string            `bson:"name"`
	Labels      map[string]string `bson:"labels,omitempty"`
	Annotations map[string]string `bson:"annotations,omitempty"`
	CreatedAt   time.Time         `bson:"createdAt"`
	DeletedAt   *time.Time        `bson:"deletedAt,omitempty"`
}

// NamespaceResourceStatus represents the status of a namespace resource.
type NamespaceResourceStatus struct {
	Conditions []Condition `bson:"conditions,omitempty"`
}

// ToDomain converts the entity to domain model.
func (ns *Namespace) ToDomain() *agentmodel.Namespace {
	return &agentmodel.Namespace{
		Metadata: ns.Metadata.toDomain(),
		Status:   ns.Status.toDomain(),
	}
}

func (m *NamespaceMetadata) toDomain() agentmodel.NamespaceMetadata {
	return agentmodel.NamespaceMetadata{
		Name:        m.Name,
		Labels:      m.Labels,
		Annotations: m.Annotations,
		CreatedAt:   m.CreatedAt,
		DeletedAt:   m.DeletedAt,
	}
}

func (s *NamespaceResourceStatus) toDomain() agentmodel.NamespaceStatus {
	return agentmodel.NamespaceStatus{
		Conditions: lo.Map(
			s.Conditions,
			func(c Condition, _ int) model.Condition {
				return c.ToDomain()
			},
		),
	}
}

// NamespaceFromDomain converts domain model to entity.
func NamespaceFromDomain(domain *agentmodel.Namespace) *Namespace {
	return &Namespace{
		Common: Common{
			Version: VersionV1,
			ID:      nil,
		},
		Metadata: namespaceMetadataFromDomain(domain.Metadata),
		Status:   namespaceStatusFromDomain(domain.Status),
	}
}

func namespaceMetadataFromDomain(
	metadata agentmodel.NamespaceMetadata,
) NamespaceMetadata {
	return NamespaceMetadata{
		Name:        metadata.Name,
		Labels:      metadata.Labels,
		Annotations: metadata.Annotations,
		CreatedAt:   metadata.CreatedAt,
		DeletedAt:   metadata.DeletedAt,
	}
}

func namespaceStatusFromDomain(
	s agentmodel.NamespaceStatus,
) NamespaceResourceStatus {
	return NamespaceResourceStatus{
		Conditions: lo.Map(
			s.Conditions,
			func(c model.Condition, _ int) Condition {
				return NewConditionFromDomain(c)
			},
		),
	}
}
