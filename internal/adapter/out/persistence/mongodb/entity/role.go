package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

const (
	// RoleKeyFieldName is the key field name for role.
	RoleKeyFieldName = "metadata.uid"
)

// Role is the MongoDB entity for role.
type Role struct {
	Common `bson:",inline"`

	Metadata RoleMetadata `bson:"metadata"`
	Spec     RoleSpec     `bson:"spec"`
	Status   RoleStatus   `bson:"status"`
}

// RoleMetadata represents the metadata of a role.
type RoleMetadata struct {
	UID       string     `bson:"uid"`
	CreatedAt time.Time  `bson:"createdAt,omitempty"`
	UpdatedAt time.Time  `bson:"updatedAt,omitempty"`
	DeletedAt *time.Time `bson:"deletedAt,omitempty"`
}

// RoleSpec represents the specification of a role.
type RoleSpec struct {
	DisplayName string   `bson:"displayName"`
	Description string   `bson:"description"`
	Permissions []string `bson:"permissions,omitempty"`
	IsBuiltIn   bool     `bson:"isBuiltIn"`
}

// RoleStatus represents the status of a role.
type RoleStatus struct {
	Conditions []Condition `bson:"conditions,omitempty"`
}

// ToDomain converts the entity to domain model.
func (r *Role) ToDomain() *model.Role {
	return &model.Role{
		Metadata: r.Metadata.toDomain(),
		Spec:     r.Spec.toDomain(),
		Status:   r.Status.toDomain(),
	}
}

func (m *RoleMetadata) toDomain() model.RoleMetadata {
	return model.RoleMetadata{
		UID:       uuid.MustParse(m.UID),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
	}
}

func (s *RoleSpec) toDomain() model.RoleSpec {
	return model.RoleSpec{
		DisplayName: s.DisplayName,
		Description: s.Description,
		Permissions: s.Permissions,
		IsBuiltIn:   s.IsBuiltIn,
	}
}

func (s *RoleStatus) toDomain() model.RoleStatus {
	return model.RoleStatus{
		Conditions: lo.Map(s.Conditions, func(c Condition, _ int) model.Condition {
			return c.ToDomain()
		}),
	}
}

// RoleFromDomain converts domain model to entity.
func RoleFromDomain(domain *model.Role) *Role {
	return &Role{
		Common: Common{
			Version: VersionV1,
			ID:      nil,
		},
		Metadata: roleMetadataFromDomain(domain.Metadata),
		Spec:     roleSpecFromDomain(domain.Spec),
		Status:   roleStatusFromDomain(domain.Status),
	}
}

func roleMetadataFromDomain(m model.RoleMetadata) RoleMetadata {
	return RoleMetadata{
		UID:       m.UID.String(),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
	}
}

func roleSpecFromDomain(s model.RoleSpec) RoleSpec {
	return RoleSpec{
		DisplayName: s.DisplayName,
		Description: s.Description,
		Permissions: s.Permissions,
		IsBuiltIn:   s.IsBuiltIn,
	}
}

func roleStatusFromDomain(s model.RoleStatus) RoleStatus {
	return RoleStatus{
		Conditions: lo.Map(s.Conditions, func(c model.Condition, _ int) Condition {
			return NewConditionFromDomain(c)
		}),
	}
}
