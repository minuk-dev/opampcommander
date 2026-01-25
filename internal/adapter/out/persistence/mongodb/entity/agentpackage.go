package entity

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"

	domainmodel "github.com/minuk-dev/opampcommander/internal/domain/model"
)

const (
	// AgentPackageKeyFieldName is the key field name for agent package.
	AgentPackageKeyFieldName = "name"
)

// AgentPackageResource is the MongoDB entity for agent package resource.
type AgentPackageResource struct {
	ID       *bson.ObjectID               `bson:"_id,omitempty"`
	Name     string                       `bson:"name"`
	Metadata AgentPackageResourceMetadata `bson:"metadata"`
	Spec     AgentPackageResourceSpec     `bson:"spec"`
	Status   AgentPackageResourceStatus   `bson:"status"`
}

// AgentPackageResourceMetadata represents the metadata of an agent package resource.
type AgentPackageResourceMetadata struct {
	Attributes map[string]string `bson:"attributes,omitempty"`
}

// AgentPackageResourceSpec represents the specification of an agent package resource.
type AgentPackageResourceSpec struct {
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
func (ap *AgentPackageResource) ToDomain() *domainmodel.AgentPackage {
	conditions := make([]domainmodel.Condition, len(ap.Status.Conditions))
	for i, c := range ap.Status.Conditions {
		conditions[i] = domainmodel.Condition{
			Type:               domainmodel.ConditionType(c.Type),
			Status:             domainmodel.ConditionStatus(c.Status),
			LastTransitionTime: c.LastTransitionTime,
			Reason:             c.Reason,
			Message:            c.Message,
		}
	}

	return &domainmodel.AgentPackage{
		Metadata: domainmodel.AgentPackageMetadata{
			Name:       ap.Name,
			Attributes: ap.Metadata.Attributes,
		},
		Spec: domainmodel.AgentPackageSpec{
			PackageType: ap.Spec.PackageType,
			Version:     ap.Spec.Version,
			DownloadURL: ap.Spec.DownloadURL,
			ContentHash: ap.Spec.ContentHash,
			Signature:   ap.Spec.Signature,
			Headers:     ap.Spec.Headers,
			Hash:        ap.Spec.Hash,
		},
		Status: domainmodel.AgentPackageStatus{
			Conditions: conditions,
		},
	}
}

// AgentPackageResourceFromDomain converts domain model to entity.
func AgentPackageResourceFromDomain(ap *domainmodel.AgentPackage) *AgentPackageResource {
	conditions := make([]Condition, len(ap.Status.Conditions))
	for i, c := range ap.Status.Conditions {
		conditions[i] = Condition{
			Type:               string(c.Type),
			Status:             string(c.Status),
			LastTransitionTime: c.LastTransitionTime,
			Reason:             c.Reason,
			Message:            c.Message,
		}
	}

	return &AgentPackageResource{
		Name: ap.Metadata.Name,
		Metadata: AgentPackageResourceMetadata{
			Attributes: ap.Metadata.Attributes,
		},
		Spec: AgentPackageResourceSpec{
			PackageType: ap.Spec.PackageType,
			Version:     ap.Spec.Version,
			DownloadURL: ap.Spec.DownloadURL,
			ContentHash: ap.Spec.ContentHash,
			Signature:   ap.Spec.Signature,
			Headers:     ap.Spec.Headers,
			Hash:        ap.Spec.Hash,
		},
		Status: AgentPackageResourceStatus{
			Conditions: conditions,
		},
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
	Attributes     map[string]string `bson:"attributes,omitempty"`
	CreatedAtMilli int64             `bson:"createdAtMilli,omitempty"`
	CreatedBy      string            `bson:"createdBy,omitempty"`
	UpdatedAtMilli int64             `bson:"updatedAtMilli,omitempty"`
	UpdatedBy      string            `bson:"updatedBy,omitempty"`
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
			CreatedAt:  time.UnixMilli(arc.Metadata.CreatedAtMilli),
			CreatedBy:  arc.Metadata.CreatedBy,
			UpdatedAt:  time.UnixMilli(arc.Metadata.UpdatedAtMilli),
			UpdatedBy:  arc.Metadata.UpdatedBy,
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
			Attributes:     arc.Metadata.Attributes,
			CreatedAtMilli: arc.Metadata.CreatedAt.UnixMilli(),
			CreatedBy:      arc.Metadata.CreatedBy,
			UpdatedAtMilli: arc.Metadata.UpdatedAt.UnixMilli(),
			UpdatedBy:      arc.Metadata.UpdatedBy,
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
