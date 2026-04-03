package entity

import (
	"time"

	"github.com/google/uuid"

	usermodel "github.com/minuk-dev/opampcommander/internal/domain/user/model"
)

const (
	// OrgRoleMappingKeyFieldName is the key field name for org role mapping.
	OrgRoleMappingKeyFieldName = "metadata.uid"
)

// OrgRoleMapping is the MongoDB entity for org-role mapping.
type OrgRoleMapping struct {
	Common `bson:",inline"`

	Metadata OrgRoleMappingMetadata `bson:"metadata"`
	Spec     OrgRoleMappingSpec     `bson:"spec"`
}

// OrgRoleMappingMetadata represents the metadata of an org-role mapping.
type OrgRoleMappingMetadata struct {
	UID       string     `bson:"uid"`
	CreatedAt time.Time  `bson:"createdAt,omitempty"`
	UpdatedAt time.Time  `bson:"updatedAt,omitempty"`
	DeletedAt *time.Time `bson:"deletedAt,omitempty"`
}

// OrgRoleMappingSpec represents the specification of an org-role mapping.
type OrgRoleMappingSpec struct {
	Provider     string `bson:"provider"`
	Organization string `bson:"organization"`
	Team         string `bson:"team,omitempty"`
	RoleID       string `bson:"roleId"`
}

// ToDomain converts the entity to domain model.
func (o *OrgRoleMapping) ToDomain() *usermodel.OrgRoleMapping {
	return &usermodel.OrgRoleMapping{
		Metadata: o.Metadata.toDomain(),
		Spec:     o.Spec.toDomain(),
	}
}

func (m *OrgRoleMappingMetadata) toDomain() usermodel.OrgRoleMappingMetadata {
	return usermodel.OrgRoleMappingMetadata{
		UID:       uuid.MustParse(m.UID),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
	}
}

func (s *OrgRoleMappingSpec) toDomain() usermodel.OrgRoleMappingSpec {
	return usermodel.OrgRoleMappingSpec{
		Provider:     s.Provider,
		Organization: s.Organization,
		Team:         s.Team,
		RoleID:       uuid.MustParse(s.RoleID),
	}
}

// OrgRoleMappingFromDomain converts domain model to entity.
func OrgRoleMappingFromDomain(domain *usermodel.OrgRoleMapping) *OrgRoleMapping {
	return &OrgRoleMapping{
		Common: Common{
			Version: VersionV1,
			ID:      nil,
		},
		Metadata: orgRoleMappingMetadataFromDomain(domain.Metadata),
		Spec:     orgRoleMappingSpecFromDomain(domain.Spec),
	}
}

func orgRoleMappingMetadataFromDomain(m usermodel.OrgRoleMappingMetadata) OrgRoleMappingMetadata {
	return OrgRoleMappingMetadata{
		UID:       m.UID.String(),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
	}
}

func orgRoleMappingSpecFromDomain(s usermodel.OrgRoleMappingSpec) OrgRoleMappingSpec {
	return OrgRoleMappingSpec{
		Provider:     s.Provider,
		Organization: s.Organization,
		Team:         s.Team,
		RoleID:       s.RoleID.String(),
	}
}
