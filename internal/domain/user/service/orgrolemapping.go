package userservice

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	usermodel "github.com/minuk-dev/opampcommander/internal/domain/user/model"
	userport "github.com/minuk-dev/opampcommander/internal/domain/user/port"
)

var _ userport.OrgRoleMappingUsecase = (*OrgRoleMappingService)(nil)

// OrgRoleMappingService manages mappings between external org/group memberships and internal roles.
type OrgRoleMappingService struct {
	orgRoleMappingPersistencePort userport.OrgRoleMappingPersistencePort
	rolePersistencePort           userport.RolePersistencePort
	logger                        *slog.Logger
}

// NewOrgRoleMappingService creates a new instance of OrgRoleMappingService.
func NewOrgRoleMappingService(
	orgRoleMappingPersistencePort userport.OrgRoleMappingPersistencePort,
	rolePersistencePort userport.RolePersistencePort,
	logger *slog.Logger,
) *OrgRoleMappingService {
	return &OrgRoleMappingService{
		orgRoleMappingPersistencePort: orgRoleMappingPersistencePort,
		rolePersistencePort:           rolePersistencePort,
		logger:                        logger,
	}
}

// GetOrgRoleMapping implements [userport.OrgRoleMappingUsecase].
func (s *OrgRoleMappingService) GetOrgRoleMapping(
	ctx context.Context,
	uid uuid.UUID,
) (*usermodel.OrgRoleMapping, error) {
	mapping, err := s.orgRoleMappingPersistencePort.GetOrgRoleMapping(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to get org-role mapping: %w", err)
	}

	return mapping, nil
}

// ListOrgRoleMappings implements [userport.OrgRoleMappingUsecase].
func (s *OrgRoleMappingService) ListOrgRoleMappings(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*usermodel.OrgRoleMapping], error) {
	resp, err := s.orgRoleMappingPersistencePort.ListOrgRoleMappings(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list org-role mappings: %w", err)
	}

	return resp, nil
}

// ListOrgRoleMappingsByProvider implements [userport.OrgRoleMappingUsecase].
func (s *OrgRoleMappingService) ListOrgRoleMappingsByProvider(
	ctx context.Context,
	provider string,
) ([]*usermodel.OrgRoleMapping, error) {
	mappings, err := s.orgRoleMappingPersistencePort.ListOrgRoleMappingsByProvider(ctx, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to list org-role mappings by provider: %w", err)
	}

	return mappings, nil
}

// SaveOrgRoleMapping implements [userport.OrgRoleMappingUsecase].
func (s *OrgRoleMappingService) SaveOrgRoleMapping(
	ctx context.Context,
	mapping *usermodel.OrgRoleMapping,
) error {
	_, err := s.orgRoleMappingPersistencePort.PutOrgRoleMapping(ctx, mapping)
	if err != nil {
		return fmt.Errorf("failed to save org-role mapping: %w", err)
	}

	return nil
}

// DeleteOrgRoleMapping implements [userport.OrgRoleMappingUsecase].
func (s *OrgRoleMappingService) DeleteOrgRoleMapping(
	ctx context.Context,
	uid uuid.UUID,
) error {
	err := s.orgRoleMappingPersistencePort.DeleteOrgRoleMapping(ctx, uid)
	if err != nil {
		return fmt.Errorf("failed to delete org-role mapping: %w", err)
	}

	return nil
}

// ResolveRolesForIdentity resolves which roles should be assigned based on
// an external identity's org/group memberships and the configured mappings.
func (s *OrgRoleMappingService) ResolveRolesForIdentity(
	ctx context.Context,
	identity *usermodel.ExternalIdentity,
) ([]*usermodel.Role, error) {
	if identity == nil {
		return nil, nil
	}

	mappings, err := s.orgRoleMappingPersistencePort.ListOrgRoleMappingsByProvider(ctx, identity.Provider)
	if err != nil {
		return nil, fmt.Errorf("failed to list org-role mappings: %w", err)
	}

	// Collect unique role IDs from matching mappings
	roleIDSet := make(map[uuid.UUID]struct{})

	for _, mapping := range mappings {
		for _, group := range identity.Groups {
			if mapping.Matches(identity.Provider, group, "") {
				roleIDSet[mapping.Spec.RoleID] = struct{}{}
			}
		}
	}

	// Resolve role IDs to full Role models
	roles := make([]*usermodel.Role, 0, len(roleIDSet))

	for roleID := range roleIDSet {
		role, err := s.rolePersistencePort.GetRole(ctx, roleID)
		if err != nil {
			s.logger.Warn("failed to resolve role for org mapping",
				slog.String("roleID", roleID.String()),
				slog.String("error", err.Error()),
			)

			continue
		}

		roles = append(roles, role)
	}

	return roles, nil
}
