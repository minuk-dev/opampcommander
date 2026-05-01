package entity

import (
	"maps"
	"time"

	"github.com/samber/lo"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	usermodel "github.com/minuk-dev/opampcommander/internal/domain/user/model"
)

const (
	// RoleBindingKeyFieldName is the key field name for role binding entities in MongoDB.
	RoleBindingKeyFieldName string = "metadata.name"
)

// RoleBinding is the MongoDB entity for a RoleBinding resource.
type RoleBinding struct {
	Common `bson:",inline"`

	Metadata RoleBindingMetadata `bson:"metadata"`
	Spec     RoleBindingSpec     `bson:"spec"`
	Status   RoleBindingStatus   `bson:"status"`
}

// RoleBindingMetadata represents the metadata of a role binding.
type RoleBindingMetadata struct {
	Namespace string     `bson:"namespace"`
	Name      string     `bson:"name"`
	CreatedAt time.Time  `bson:"createdAt"`
	UpdatedAt time.Time  `bson:"updatedAt"`
	DeletedAt *time.Time `bson:"deletedAt,omitempty"`
}

// RoleBindingSpec represents the specification of a role binding.
type RoleBindingSpec struct {
	RoleRef       RoleBindingRoleRef `bson:"roleRef"`
	LabelSelector map[string]string  `bson:"labelSelector,omitempty"`
}

// RoleBindingRoleRef references a role in MongoDB.
type RoleBindingRoleRef struct {
	Kind string `bson:"kind"`
	Name string `bson:"name"`
}

// RoleBindingStatus represents the status of a role binding.
type RoleBindingStatus struct {
	Conditions []Condition `bson:"conditions,omitempty"`
}

// ToDomain converts the entity to domain model.
func (e *RoleBinding) ToDomain() *usermodel.RoleBinding {
	return &usermodel.RoleBinding{
		Metadata: e.Metadata.toDomain(),
		Spec:     e.Spec.toDomain(),
		Status:   e.Status.toDomain(),
	}
}

func (m *RoleBindingMetadata) toDomain() usermodel.RoleBindingMetadata {
	return usermodel.RoleBindingMetadata{
		Namespace: m.Namespace,
		Name:      m.Name,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
	}
}

func (s *RoleBindingSpec) toDomain() usermodel.RoleBindingSpec {
	labelSelector := make(map[string]string, len(s.LabelSelector))
	maps.Copy(labelSelector, s.LabelSelector)

	return usermodel.RoleBindingSpec{
		RoleRef: usermodel.RoleRef{
			Kind: s.RoleRef.Kind,
			Name: s.RoleRef.Name,
		},
		LabelSelector: labelSelector,
	}
}

func (s *RoleBindingStatus) toDomain() usermodel.RoleBindingStatus {
	return usermodel.RoleBindingStatus{
		Conditions: lo.Map(s.Conditions, func(c Condition, _ int) model.Condition {
			return c.ToDomain()
		}),
	}
}

// RoleBindingFromDomain converts domain model to entity.
func RoleBindingFromDomain(domain *usermodel.RoleBinding) *RoleBinding {
	return &RoleBinding{
		Common: Common{
			Version: VersionV1,
			ID:      nil,
		},
		Metadata: roleBindingMetadataFromDomain(domain.Metadata),
		Spec:     roleBindingSpecFromDomain(domain.Spec),
		Status:   roleBindingStatusFromDomain(domain.Status),
	}
}

func roleBindingMetadataFromDomain(metadata usermodel.RoleBindingMetadata) RoleBindingMetadata {
	return RoleBindingMetadata{
		Namespace: metadata.Namespace,
		Name:      metadata.Name,
		CreatedAt: metadata.CreatedAt,
		UpdatedAt: metadata.UpdatedAt,
		DeletedAt: metadata.DeletedAt,
	}
}

func roleBindingSpecFromDomain(spec usermodel.RoleBindingSpec) RoleBindingSpec {
	labelSelector := make(map[string]string, len(spec.LabelSelector))
	maps.Copy(labelSelector, spec.LabelSelector)

	return RoleBindingSpec{
		RoleRef: RoleBindingRoleRef{
			Kind: spec.RoleRef.Kind,
			Name: spec.RoleRef.Name,
		},
		LabelSelector: labelSelector,
	}
}

func roleBindingStatusFromDomain(s usermodel.RoleBindingStatus) RoleBindingStatus {
	return RoleBindingStatus{
		Conditions: lo.Map(s.Conditions, func(c model.Condition, _ int) Condition {
			return NewConditionFromDomain(c)
		}),
	}
}
