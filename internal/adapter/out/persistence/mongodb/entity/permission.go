package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

const (
	// PermissionKeyFieldName is the key field name for permission.
	PermissionKeyFieldName = "metadata.uid"
)

// Permission is the MongoDB entity for permission.
type Permission struct {
	Common `bson:",inline"`

	Metadata PermissionMetadata `bson:"metadata"`
	Spec     PermissionSpec     `bson:"spec"`
	Status   PermissionStatus   `bson:"status"`
}

// PermissionMetadata represents the metadata of a permission.
type PermissionMetadata struct {
	UID       string     `bson:"uid"`
	CreatedAt time.Time  `bson:"createdAt,omitempty"`
	UpdatedAt time.Time  `bson:"updatedAt,omitempty"`
	DeletedAt *time.Time `bson:"deletedAt,omitempty"`
}

// PermissionSpec represents the specification of a permission.
type PermissionSpec struct {
	Name        string `bson:"name"`
	Description string `bson:"description"`
	Resource    string `bson:"resource"`
	Action      string `bson:"action"`
	IsBuiltIn   bool   `bson:"isBuiltIn"`
}

// PermissionStatus represents the status of a permission.
type PermissionStatus struct {
	Conditions []Condition `bson:"conditions,omitempty"`
}

// ToDomain converts the entity to domain model.
func (p *Permission) ToDomain() *model.Permission {
	return &model.Permission{
		Metadata: p.Metadata.toDomain(),
		Spec:     p.Spec.toDomain(),
		Status:   p.Status.toDomain(),
	}
}

func (m *PermissionMetadata) toDomain() model.PermissionMetadata {
	return model.PermissionMetadata{
		UID:       uuid.MustParse(m.UID),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
	}
}

func (s *PermissionSpec) toDomain() model.PermissionSpec {
	return model.PermissionSpec{
		Name:        s.Name,
		Description: s.Description,
		Resource:    s.Resource,
		Action:      s.Action,
		IsBuiltIn:   s.IsBuiltIn,
	}
}

func (s *PermissionStatus) toDomain() model.PermissionStatus {
	return model.PermissionStatus{
		Conditions: lo.Map(s.Conditions, func(c Condition, _ int) model.Condition {
			return c.ToDomain()
		}),
	}
}

// PermissionFromDomain converts domain model to entity.
func PermissionFromDomain(domain *model.Permission) *Permission {
	return &Permission{
		Common: Common{
			Version: VersionV1,
			ID:      nil,
		},
		Metadata: permissionMetadataFromDomain(domain.Metadata),
		Spec:     permissionSpecFromDomain(domain.Spec),
		Status:   permissionStatusFromDomain(domain.Status),
	}
}

func permissionMetadataFromDomain(m model.PermissionMetadata) PermissionMetadata {
	return PermissionMetadata{
		UID:       m.UID.String(),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
	}
}

func permissionSpecFromDomain(s model.PermissionSpec) PermissionSpec {
	return PermissionSpec{
		Name:        s.Name,
		Description: s.Description,
		Resource:    s.Resource,
		Action:      s.Action,
		IsBuiltIn:   s.IsBuiltIn,
	}
}

func permissionStatusFromDomain(s model.PermissionStatus) PermissionStatus {
	return PermissionStatus{
		Conditions: lo.Map(s.Conditions, func(c model.Condition, _ int) Condition {
			return NewConditionFromDomain(c)
		}),
	}
}
