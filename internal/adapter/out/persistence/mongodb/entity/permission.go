package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	usermodel "github.com/minuk-dev/opampcommander/internal/domain/user/model"
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
func (p *Permission) ToDomain() *usermodel.Permission {
	return &usermodel.Permission{
		Metadata: p.Metadata.toDomain(),
		Spec:     p.Spec.toDomain(),
		Status:   p.Status.toDomain(),
	}
}

func (m *PermissionMetadata) toDomain() usermodel.PermissionMetadata {
	return usermodel.PermissionMetadata{
		UID:       uuid.MustParse(m.UID),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
	}
}

func (s *PermissionSpec) toDomain() usermodel.PermissionSpec {
	return usermodel.PermissionSpec{
		Name:        s.Name,
		Description: s.Description,
		Resource:    s.Resource,
		Action:      s.Action,
		IsBuiltIn:   s.IsBuiltIn,
	}
}

func (s *PermissionStatus) toDomain() usermodel.PermissionStatus {
	return usermodel.PermissionStatus{
		Conditions: lo.Map(s.Conditions, func(c Condition, _ int) model.Condition {
			return c.ToDomain()
		}),
	}
}

// PermissionFromDomain converts domain model to entity.
func PermissionFromDomain(domain *usermodel.Permission) *Permission {
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

func permissionMetadataFromDomain(m usermodel.PermissionMetadata) PermissionMetadata {
	return PermissionMetadata{
		UID:       m.UID.String(),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
	}
}

func permissionSpecFromDomain(spec usermodel.PermissionSpec) PermissionSpec {
	return PermissionSpec{
		Name:        spec.Name,
		Description: spec.Description,
		Resource:    spec.Resource,
		Action:      spec.Action,
		IsBuiltIn:   spec.IsBuiltIn,
	}
}

func permissionStatusFromDomain(s usermodel.PermissionStatus) PermissionStatus {
	return PermissionStatus{
		Conditions: lo.Map(s.Conditions, func(c model.Condition, _ int) Condition {
			return NewConditionFromDomain(c)
		}),
	}
}
