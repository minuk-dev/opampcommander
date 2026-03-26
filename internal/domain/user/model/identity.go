package usermodel

import (
	"time"

	"github.com/google/uuid"
)

// IdentityProvider constants for supported authentication providers.
const (
	IdentityProviderGitHub = "github"
	IdentityProviderGoogle = "google"
	IdentityProviderLDAP   = "ldap"
	IdentityProviderOIDC   = "oidc"
	IdentityProviderBasic  = "basic"
)

// ExternalIdentity represents the identity information retrieved from an external provider.
// It is provider-agnostic and used as the common interface between authentication and RBAC.
type ExternalIdentity struct {
	Provider       string
	ProviderUserID string
	Email          string
	DisplayName    string
	AvatarURL      string
	Groups         []string          // org/team memberships from provider (e.g., GitHub orgs)
	RawAttributes  map[string]string // provider-specific metadata
}

// OrgRoleMapping defines a mapping from an external provider's organization/group
// to an internal RBAC role. This enables automatic role assignment based on
// provider group memberships (e.g., GitHub org → Admin role).
type OrgRoleMapping struct {
	Metadata OrgRoleMappingMetadata
	Spec     OrgRoleMappingSpec
}

// OrgRoleMappingMetadata contains metadata about the org-role mapping.
type OrgRoleMappingMetadata struct {
	UID       uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// OrgRoleMappingSpec defines the mapping specification.
type OrgRoleMappingSpec struct {
	Provider     string // e.g., "github", "google"
	Organization string // e.g., "my-github-org"
	Team         string // optional: e.g., "platform-team" (empty means entire org)
	RoleID       uuid.UUID
}

// NewOrgRoleMapping creates a new organization-to-role mapping.
func NewOrgRoleMapping(provider, organization, team string, roleID uuid.UUID) *OrgRoleMapping {
	now := time.Now()

	return &OrgRoleMapping{
		Metadata: OrgRoleMappingMetadata{
			UID:       uuid.New(),
			CreatedAt: now,
			UpdatedAt: now,
			DeletedAt: nil,
		},
		Spec: OrgRoleMappingSpec{
			Provider:     provider,
			Organization: organization,
			Team:         team,
			RoleID:       roleID,
		},
	}
}

// Matches returns true if the given provider/org/team matches this mapping.
func (m *OrgRoleMapping) Matches(provider, org, team string) bool {
	if m.Spec.Provider != provider {
		return false
	}

	if m.Spec.Organization != org {
		return false
	}

	// If the mapping has no team restriction, any team matches
	if m.Spec.Team == "" {
		return true
	}

	return m.Spec.Team == team
}

// Delete marks the org-role mapping as deleted.
func (m *OrgRoleMapping) Delete() {
	now := time.Now()
	m.Metadata.DeletedAt = &now
}
