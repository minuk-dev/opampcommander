package entity

import (
	"time"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
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
func (o *OrgRoleMapping) ToDomain() *model.OrgRoleMapping {
	return &model.OrgRoleMapping{
		Metadata: o.Metadata.toDomain(),
		Spec:     o.Spec.toDomain(),
	}
}

func (m *OrgRoleMappingMetadata) toDomain() model.OrgRoleMappingMetadata {
	return model.OrgRoleMappingMetadata{
		UID:       uuid.MustParse(m.UID),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
	}
}

func (s *OrgRoleMappingSpec) toDomain() model.OrgRoleMappingSpec {
	return model.OrgRoleMappingSpec{
		Provider:     s.Provider,
		Organization: s.Organization,
		Team:         s.Team,
		RoleID:       uuid.MustParse(s.RoleID),
	}
}

// OrgRoleMappingFromDomain converts domain model to entity.
func OrgRoleMappingFromDomain(domain *model.OrgRoleMapping) *OrgRoleMapping {
	return &OrgRoleMapping{
		Common: Common{
			Version: VersionV1,
			ID:      nil,
		},
		Metadata: orgRoleMappingMetadataFromDomain(domain.Metadata),
		Spec:     orgRoleMappingSpecFromDomain(domain.Spec),
	}
}

func orgRoleMappingMetadataFromDomain(m model.OrgRoleMappingMetadata) OrgRoleMappingMetadata {
	return OrgRoleMappingMetadata{
		UID:       m.UID.String(),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
	}
}

func orgRoleMappingSpecFromDomain(s model.OrgRoleMappingSpec) OrgRoleMappingSpec {
	return OrgRoleMappingSpec{
		Provider:     s.Provider,
		Organization: s.Organization,
		Team:         s.Team,
		RoleID:       s.RoleID.String(),
	}
}
