package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

// ErrBuiltInRoleCannotBeDeleted is returned when attempting to delete a built-in role.
var ErrBuiltInRoleCannotBeDeleted = errors.New("built-in role cannot be deleted")

var _ port.RoleUsecase = (*RoleService)(nil)

// RoleService implements the RoleUsecase interface.
type RoleService struct {
	rolePersistencePort port.RolePersistencePort
	logger              *slog.Logger
}

// NewRoleService creates a new instance of RoleService.
func NewRoleService(
	rolePersistencePort port.RolePersistencePort,
	logger *slog.Logger,
) *RoleService {
	return &RoleService{
		rolePersistencePort: rolePersistencePort,
		logger:              logger,
	}
}

// GetRole implements [port.RoleUsecase].
func (s *RoleService) GetRole(
	ctx context.Context,
	uid uuid.UUID,
) (*model.Role, error) {
	role, err := s.rolePersistencePort.GetRole(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to get role from persistence: %w", err)
	}

	return role, nil
}

// GetRoleByName implements [port.RoleUsecase].
func (s *RoleService) GetRoleByName(
	ctx context.Context,
	displayName string,
) (*model.Role, error) {
	role, err := s.rolePersistencePort.GetRoleByName(ctx, displayName)
	if err != nil {
		return nil, fmt.Errorf("failed to get role by name from persistence: %w", err)
	}

	return role, nil
}

// ListRoles implements [port.RoleUsecase].
func (s *RoleService) ListRoles(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*model.Role], error) {
	resp, err := s.rolePersistencePort.ListRoles(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles from persistence: %w", err)
	}

	return resp, nil
}

// SaveRole implements [port.RoleUsecase].
func (s *RoleService) SaveRole(
	ctx context.Context,
	role *model.Role,
) error {
	_, err := s.rolePersistencePort.PutRole(ctx, role)
	if err != nil {
		return fmt.Errorf("failed to save role to persistence: %w", err)
	}

	return nil
}

// DeleteRole implements [port.RoleUsecase].
func (s *RoleService) DeleteRole(
	ctx context.Context,
	uid uuid.UUID,
) error {
	role, err := s.rolePersistencePort.GetRole(ctx, uid)
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
