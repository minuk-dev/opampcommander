package entity

import (
	"time"

	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
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
	CreatedAt  time.Time         `bson:"createdAt"`
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
func (ap *AgentPackage) ToDomain() *agentmodel.AgentPackage {
	return &agentmodel.AgentPackage{
		Metadata: ap.Metadata.toDomain(),
		Spec:     ap.Spec.toDomain(),
		Status:   ap.Status.toDomain(),
	}
}

func (m *AgentPackageMetadata) toDomain() agentmodel.AgentPackageMetadata {
	return agentmodel.AgentPackageMetadata{
		Name:       m.Name,
		Attributes: m.Attributes,
		CreatedAt:  m.CreatedAt,
		DeletedAt:  m.DeletedAt,
	}
}

func (s *AgentPackageSpec) toDomain() agentmodel.AgentPackageSpec {
	return agentmodel.AgentPackageSpec{
		PackageType: s.PackageType,
		Version:     s.Version,
		DownloadURL: s.DownloadURL,
		ContentHash: s.ContentHash,
		Signature:   s.Signature,
		Headers:     s.Headers,
		Hash:        s.Hash,
	}
}

func (s *AgentPackageResourceStatus) toDomain() agentmodel.AgentPackageStatus {
	return agentmodel.AgentPackageStatus{
		Conditions: lo.Map(s.Conditions, func(c Condition, _ int) model.Condition {
			return c.ToDomain()
		}),
	}
}

// AgentPackageFromDomain converts domain model to entity.
func AgentPackageFromDomain(domain *agentmodel.AgentPackage) *AgentPackage {
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

func agentPackageMetadataFromDomain(m agentmodel.AgentPackageMetadata) AgentPackageMetadata {
	return AgentPackageMetadata{
		Name:       m.Name,
		Attributes: m.Attributes,
		CreatedAt:  m.CreatedAt,
		DeletedAt:  m.DeletedAt,
	}
}

func agentPackageSpecFromDomain(domain agentmodel.AgentPackageSpec) AgentPackageSpec {
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

func agentPackageStatusFromDomain(s agentmodel.AgentPackageStatus) AgentPackageResourceStatus {
	return AgentPackageResourceStatus{
		Conditions: lo.Map(s.Conditions, func(c model.Condition, _ int) Condition {
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
	CreatedAt  time.Time         `bson:"createdAt"`
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
func (arc *AgentRemoteConfigResourceEntity) ToDomain() *agentmodel.AgentRemoteConfig {
	return &agentmodel.AgentRemoteConfig{
		Metadata: agentmodel.AgentRemoteConfigMetadata{
			Name:       arc.Name,
			Attributes: arc.Metadata.Attributes,
			CreatedAt:  arc.Metadata.CreatedAt,
			DeletedAt:  nil,
		},
		Spec: agentmodel.AgentRemoteConfigSpec{
			Value:       arc.Spec.Value,
			ContentType: arc.Spec.ContentType,
		},
		Status: agentmodel.AgentRemoteConfigResourceStatus{
			Conditions: lo.Map(arc.Status.Conditions, func(c Condition, _ int) model.Condition {
				return c.ToDomain()
			}),
		},
	}
}

// AgentRemoteConfigResourceEntityFromDomain converts domain model to entity.
func AgentRemoteConfigResourceEntityFromDomain(
	arc *agentmodel.AgentRemoteConfig,
) *AgentRemoteConfigResourceEntity {
	//nolint:exhaustruct // ID is set by MongoDB
	return &AgentRemoteConfigResourceEntity{
		Name: arc.Metadata.Name,
		Metadata: AgentRemoteConfigResourceMetadata{
			Attributes: arc.Metadata.Attributes,
			CreatedAt:  arc.Metadata.CreatedAt,
		},
		Spec: AgentRemoteConfigResourceSpec{
			Value:       arc.Spec.Value,
			ContentType: arc.Spec.ContentType,
		},
		Status: AgentRemoteConfigResourceEntityStatus{
			Conditions: lo.Map(arc.Status.Conditions, func(c model.Condition, _ int) Condition {
				return NewConditionFromDomain(c)
			}),
		},
	}
}
