package entity

import (
	"time"

	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	domainmodel "github.com/minuk-dev/opampcommander/internal/domain/model"
)

const (
	// AgentPackageKeyFieldName is the key field name for agent package.
	AgentPackageKeyFieldName = "metadata.name"
)

// AgentPackage is the MongoDB entity for agent package.
type AgentPackage struct {
	Common `bson:",inline"`

	Metadata AgentPackageMetadata       `bson:"metadata"`
	Spec     AgentPackageSpec           `bson:"spec"`
	Status   AgentPackageResourceStatus `bson:"status"`
}

// AgentPackageMetadata represents the metadata of an agent package.
type AgentPackageMetadata struct {
	Name       string            `bson:"name"`
	Attributes map[string]string `bson:"attributes,omitempty"`
	DeletedAt  *time.Time        `bson:"deletedAt,omitempty"`
}

// AgentPackageSpec represents the specification of an agent package.
type AgentPackageSpec struct {
	PackageType string            `bson:"packageType"`
	Version     string            `bson:"version"`
	DownloadURL string            `bson:"downloadUrl"`
	ContentHash []byte            `bson:"contentHash,omitempty"`
	Signature   []byte            `bson:"signature,omitempty"`
	Headers     map[string]string `bson:"headers,omitempty"`
	Hash        []byte            `bson:"hash,omitempty"`
}

// AgentPackageResourceStatus represents the status of an agent package resource.
type AgentPackageResourceStatus struct {
	Conditions []Condition `bson:"conditions,omitempty"`
}

// ToDomain converts the entity to domain model.
func (ap *AgentPackage) ToDomain() *domainmodel.AgentPackage {
	return &domainmodel.AgentPackage{
		Metadata: ap.Metadata.toDomain(),
		Spec:     ap.Spec.toDomain(),
		Status:   ap.Status.toDomain(),
	}
}

func (m *AgentPackageMetadata) toDomain() domainmodel.AgentPackageMetadata {
	return domainmodel.AgentPackageMetadata{
		Name:       m.Name,
		Attributes: m.Attributes,
		DeletedAt:  m.DeletedAt,
	}
}

func (s *AgentPackageSpec) toDomain() domainmodel.AgentPackageSpec {
	return domainmodel.AgentPackageSpec{
		PackageType: s.PackageType,
		Version:     s.Version,
		DownloadURL: s.DownloadURL,
		ContentHash: s.ContentHash,
		Signature:   s.Signature,
		Headers:     s.Headers,
		Hash:        s.Hash,
	}
}

func (s *AgentPackageResourceStatus) toDomain() domainmodel.AgentPackageStatus {
	return domainmodel.AgentPackageStatus{
		Conditions: lo.Map(s.Conditions, func(c Condition, _ int) domainmodel.Condition {
			return c.ToDomain()
		}),
	}
}

// AgentPackageFromDomain converts domain model to entity.
func AgentPackageFromDomain(domain *domainmodel.AgentPackage) *AgentPackage {
	return &AgentPackage{
		Common: Common{
			Version: VersionV1,
			ID:      nil,
		},
		Metadata: agentPackageMetadataFromDomain(domain.Metadata),
		Spec:     agentPackageSpecFromDomain(domain.Spec),
		Status:   agentPackageStatusFromDomain(domain.Status),
	}
}

func agentPackageMetadataFromDomain(m domainmodel.AgentPackageMetadata) AgentPackageMetadata {
	return AgentPackageMetadata{
		Name:       m.Name,
		Attributes: m.Attributes,
		DeletedAt:  m.DeletedAt,
	}
}

func agentPackageSpecFromDomain(domain domainmodel.AgentPackageSpec) AgentPackageSpec {
	return AgentPackageSpec{
		PackageType: domain.PackageType,
		Version:     domain.Version,
		DownloadURL: domain.DownloadURL,
		ContentHash: domain.ContentHash,
		Signature:   domain.Signature,
		Headers:     domain.Headers,
		Hash:        domain.Hash,
	}
}

func agentPackageStatusFromDomain(s domainmodel.AgentPackageStatus) AgentPackageResourceStatus {
	return AgentPackageResourceStatus{
		Conditions: lo.Map(s.Conditions, func(c domainmodel.Condition, _ int) Condition {
			return NewConditionFromDomain(c)
		}),
	}
}

const (
	// AgentRemoteConfigKeyFieldName is the key field name for agent remote config.
	AgentRemoteConfigKeyFieldName = "name"
)

// AgentRemoteConfigResourceEntity is the MongoDB entity for agent remote config resource.
type AgentRemoteConfigResourceEntity struct {
	ID       *bson.ObjectID                        `bson:"_id,omitempty"`
	Name     string                                `bson:"name"`
	Metadata AgentRemoteConfigResourceMetadata     `bson:"metadata"`
	Spec     AgentRemoteConfigResourceSpec         `bson:"spec"`
	Status   AgentRemoteConfigResourceEntityStatus `bson:"status"`
}

// AgentRemoteConfigResourceMetadata represents the metadata of an agent remote config resource.
type AgentRemoteConfigResourceMetadata struct {
	Attributes map[string]string `bson:"attributes,omitempty"`
}

// AgentRemoteConfigResourceSpec represents the specification of an agent remote config resource.
type AgentRemoteConfigResourceSpec struct {
	Value       []byte `bson:"value"`
	ContentType string `bson:"contentType"`
}

// AgentRemoteConfigResourceEntityStatus represents the status of an agent remote config resource.
type AgentRemoteConfigResourceEntityStatus struct {
	Conditions []Condition `bson:"conditions,omitempty"`
}

// ToDomain converts the entity to domain model.
func (arc *AgentRemoteConfigResourceEntity) ToDomain() *domainmodel.AgentRemoteConfig {
	return &domainmodel.AgentRemoteConfig{
		Metadata: domainmodel.AgentRemoteConfigMetadata{
			Name:       arc.Name,
			Attributes: arc.Metadata.Attributes,
			DeletedAt:  nil,
		},
		Spec: domainmodel.AgentRemoteConfigSpec{
			Value:       arc.Spec.Value,
			ContentType: arc.Spec.ContentType,
		},
		Status: domainmodel.AgentRemoteConfigResourceStatus{
			Conditions: lo.Map(arc.Status.Conditions, func(c Condition, _ int) domainmodel.Condition {
				return c.ToDomain()
			}),
		},
	}
}

// AgentRemoteConfigResourceEntityFromDomain converts domain model to entity.
func AgentRemoteConfigResourceEntityFromDomain(
	arc *domainmodel.AgentRemoteConfig,
) *AgentRemoteConfigResourceEntity {
	//nolint:exhaustruct // ID is set by MongoDB
	return &AgentRemoteConfigResourceEntity{
		Name: arc.Metadata.Name,
		Metadata: AgentRemoteConfigResourceMetadata{
			Attributes: arc.Metadata.Attributes,
		},
		Spec: AgentRemoteConfigResourceSpec{
			Value:       arc.Spec.Value,
			ContentType: arc.Spec.ContentType,
		},
		Status: AgentRemoteConfigResourceEntityStatus{
			Conditions: lo.Map(arc.Status.Conditions, func(c domainmodel.Condition, _ int) Condition {
				return NewConditionFromDomain(c)
			}),
		},
	}
}
