package entity

import (
	"go.mongodb.org/mongo-driver/v2/bson"

	domainmodel "github.com/minuk-dev/opampcommander/internal/domain/model"
)

const (
	// AgentPackageKeyFieldName is the key field name for agent package.
	AgentPackageKeyFieldName = "metadata.name"
)

// AgentPackage is the MongoDB entity for agent package.
type AgentPackage struct {
	Common   `bson:",inline"`
	Metadata AgentPackageMetadata       `bson:"metadata"`
	Spec     AgentPackageSpec           `bson:"spec"`
	Status   AgentPackageResourceStatus `bson:"status"`
}

// AgentPackageMetadata represents the metadata of an agent package.
type AgentPackageMetadata struct {
	Name       string            `bson:"name"`
	Attributes map[string]string `bson:"attributes,omitempty"`
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
	conditions := make([]domainmodel.Condition, len(s.Conditions))
	for i, c := range s.Conditions {
		conditions[i] = domainmodel.Condition{
			Type:               domainmodel.ConditionType(c.Type),
			Status:             domainmodel.ConditionStatus(c.Status),
			LastTransitionTime: c.LastTransitionTime,
			Reason:             c.Reason,
			Message:            c.Message,
		}
	}

	return domainmodel.AgentPackageStatus{
		Conditions: conditions,
	}
}

// AgentPackageFromDomain converts domain model to entity.
func AgentPackageFromDomain(ap *domainmodel.AgentPackage) *AgentPackage {
	return &AgentPackage{
		Common: Common{
			Version: VersionV1,
			ID:      nil,
		},
		Metadata: agentPackageMetadataFromDomain(ap.Metadata),
		Spec:     agentPackageSpecFromDomain(ap.Spec),
		Status:   agentPackageStatusFromDomain(ap.Status),
	}
}

func agentPackageMetadataFromDomain(m domainmodel.AgentPackageMetadata) AgentPackageMetadata {
	return AgentPackageMetadata{
		Name:       m.Name,
		Attributes: m.Attributes,
	}
}

func agentPackageSpecFromDomain(s domainmodel.AgentPackageSpec) AgentPackageSpec {
	return AgentPackageSpec{
		PackageType: s.PackageType,
		Version:     s.Version,
		DownloadURL: s.DownloadURL,
		ContentHash: s.ContentHash,
		Signature:   s.Signature,
		Headers:     s.Headers,
		Hash:        s.Hash,
	}
}

func agentPackageStatusFromDomain(s domainmodel.AgentPackageStatus) AgentPackageResourceStatus {
	conditions := make([]Condition, len(s.Conditions))
	for i, c := range s.Conditions {
		conditions[i] = Condition{
			Type:               string(c.Type),
			Status:             string(c.Status),
			LastTransitionTime: c.LastTransitionTime,
			Reason:             c.Reason,
			Message:            c.Message,
		}
	}

	return AgentPackageResourceStatus{
		Conditions: conditions,
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
func (arc *AgentRemoteConfigResourceEntity) ToDomain() *domainmodel.AgentRemoteConfigResource {
	conditions := make([]domainmodel.Condition, len(arc.Status.Conditions))
	for i, c := range arc.Status.Conditions {
		conditions[i] = domainmodel.Condition{
			Type:               domainmodel.ConditionType(c.Type),
			Status:             domainmodel.ConditionStatus(c.Status),
			LastTransitionTime: c.LastTransitionTime,
			Reason:             c.Reason,
			Message:            c.Message,
		}
	}

	return &domainmodel.AgentRemoteConfigResource{
		Metadata: domainmodel.AgentRemoteConfigMetadata{
			Name:       arc.Name,
			Attributes: arc.Metadata.Attributes,
		},
		Spec: domainmodel.AgentRemoteConfigSpec{
			Value:       arc.Spec.Value,
			ContentType: arc.Spec.ContentType,
		},
		Status: domainmodel.AgentRemoteConfigResourceStatus{
			Conditions: conditions,
		},
	}
}

// AgentRemoteConfigResourceEntityFromDomain converts domain model to entity.
func AgentRemoteConfigResourceEntityFromDomain(arc *domainmodel.AgentRemoteConfigResource) *AgentRemoteConfigResourceEntity {
	conditions := make([]Condition, len(arc.Status.Conditions))
	for i, c := range arc.Status.Conditions {
		conditions[i] = Condition{
			Type:               string(c.Type),
			Status:             string(c.Status),
			LastTransitionTime: c.LastTransitionTime,
			Reason:             c.Reason,
			Message:            c.Message,
		}
	}

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
			Conditions: conditions,
		},
	}
}
