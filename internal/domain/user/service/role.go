package userservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	usermodel "github.com/minuk-dev/opampcommander/internal/domain/user/model"
	userport "github.com/minuk-dev/opampcommander/internal/domain/user/port"
)

// ErrBuiltInRoleCannotBeDeleted is returned when attempting to delete a built-in role.
var ErrBuiltInRoleCannotBeDeleted = errors.New("built-in role cannot be deleted")

var _ userport.RoleUsecase = (*RoleService)(nil)

// RoleService implements the RoleUsecase interface.
type RoleService struct {
	rolePersistencePort userport.RolePersistencePort
	logger              *slog.Logger
}

// NewRoleService creates a new instance of RoleService.
func NewRoleService(
	rolePersistencePort userport.RolePersistencePort,
	logger *slog.Logger,
) *RoleService {
	return &RoleService{
		rolePersistencePort: rolePersistencePort,
		logger:              logger,
	}
}

// GetRole implements [userport.RoleUsecase].
func (s *RoleService) GetRole(
	ctx context.Context,
	uid uuid.UUID,
	options *model.GetOptions,
) (*usermodel.Role, error) {
	role, err := s.rolePersistencePort.GetRole(ctx, uid, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get role from persistence: %w", err)
	}

	return role, nil
}

// GetRoleByName implements [userport.RoleUsecase].
func (s *RoleService) GetRoleByName(
	ctx context.Context,
	displayName string,
) (*usermodel.Role, error) {
	role, err := s.rolePersistencePort.GetRoleByName(ctx, displayName)
	if err != nil {
		return nil, fmt.Errorf("failed to get role by name from persistence: %w", err)
	}

	return role, nil
}

// ListRoles implements [userport.RoleUsecase].
func (s *RoleService) ListRoles(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*usermodel.Role], error) {
	resp, err := s.rolePersistencePort.ListRoles(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles from persistence: %w", err)
	}

	return resp, nil
}

// SaveRole implements [userport.RoleUsecase].
func (s *RoleService) SaveRole(
	ctx context.Context,
	role *usermodel.Role,
) error {
	_, err := s.rolePersistencePort.PutRole(ctx, role)
	if err != nil {
		return fmt.Errorf("failed to save role to persistence: %w", err)
	}

	return nil
}

// DeleteRole implements [userport.RoleUsecase].
func (s *RoleService) DeleteRole(
	ctx context.Context,
	uid uuid.UUID,
) error {
	role, err := s.rolePersistencePort.GetRole(ctx, uid, nil)
	if err != nil {
		return fmt.Errorf("failed to get role from persistence: %w", err)
	}

	if role.Spec.IsBuiltIn {
		return ErrBuiltInRoleCannotBeDeleted
	}

	err = s.rolePersistencePort.DeleteRole(ctx, uid)
	if err != nil {
		return fmt.Errorf("failed to delete role from persistence: %w", err)
	}

	return nil
}
