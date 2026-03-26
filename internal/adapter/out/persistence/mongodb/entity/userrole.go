package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

const (
	// UserRoleKeyFieldName is the key field name for user role.
	UserRoleKeyFieldName = "metadata.uid"
)

// UserRole is the MongoDB entity for user role assignment.
type UserRole struct {
	Common `bson:",inline"`

	Metadata UserRoleMetadata `bson:"metadata"`
	Spec     UserRoleSpec     `bson:"spec"`
	Status   UserRoleStatus   `bson:"status"`
}

// UserRoleMetadata represents the metadata of a user role assignment.
type UserRoleMetadata struct {
	UID       string     `bson:"uid"`
	CreatedAt time.Time  `bson:"createdAt,omitempty"`
	UpdatedAt time.Time  `bson:"updatedAt,omitempty"`
	DeletedAt *time.Time `bson:"deletedAt,omitempty"`
}

// UserRoleSpec represents the specification of a user role assignment.
type UserRoleSpec struct {
	UserID     string    `bson:"userID"`
	RoleID     string    `bson:"roleID"`
	AssignedAt time.Time `bson:"assignedAt"`
	AssignedBy string    `bson:"assignedBy"`
}

// UserRoleStatus represents the status of a user role assignment.
type UserRoleStatus struct {
	Conditions []Condition `bson:"conditions,omitempty"`
}

// ToDomain converts the entity to domain model.
func (ur *UserRole) ToDomain() *model.UserRole {
	return &model.UserRole{
		Metadata: ur.Metadata.toDomain(),
		Spec:     ur.Spec.toDomain(),
		Status:   ur.Status.toDomain(),
	}
}

func (m *UserRoleMetadata) toDomain() model.UserRoleMetadata {
	return model.UserRoleMetadata{
		UID:       uuid.MustParse(m.UID),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
	}
}

func (s *UserRoleSpec) toDomain() model.UserRoleSpec {
	return model.UserRoleSpec{
		UserID:     uuid.MustParse(s.UserID),
		RoleID:     uuid.MustParse(s.RoleID),
		AssignedAt: s.AssignedAt,
		AssignedBy: uuid.MustParse(s.AssignedBy),
	}
}

func (s *UserRoleStatus) toDomain() model.UserRoleStatus {
	return model.UserRoleStatus{
		Conditions: lo.Map(s.Conditions, func(c Condition, _ int) model.Condition {
			return c.ToDomain()
		}),
	}
}

// UserRoleFromDomain converts domain model to entity.
func UserRoleFromDomain(domain *model.UserRole) *UserRole {
	return &UserRole{
		Common: Common{
			Version: VersionV1,
			ID:      nil,
		},
		Metadata: userRoleMetadataFromDomain(domain.Metadata),
		Spec:     userRoleSpecFromDomain(domain.Spec),
		Status:   userRoleStatusFromDomain(domain.Status),
	}
}

func userRoleMetadataFromDomain(m model.UserRoleMetadata) UserRoleMetadata {
	return UserRoleMetadata{
		UID:       m.UID.String(),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
	}
}

func userRoleSpecFromDomain(s model.UserRoleSpec) UserRoleSpec {
	return UserRoleSpec{
		UserID:     s.UserID.String(),
		RoleID:     s.RoleID.String(),
		AssignedAt: s.AssignedAt,
		AssignedBy: s.AssignedBy.String(),
	}
}

func userRoleStatusFromDomain(s model.UserRoleStatus) UserRoleStatus {
	return UserRoleStatus{
		Conditions: lo.Map(s.Conditions, func(c model.Condition, _ int) Condition {
			return NewConditionFromDomain(c)
		}),
	}
}
