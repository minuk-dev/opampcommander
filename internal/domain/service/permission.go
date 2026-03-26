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

// ErrBuiltInPermissionCannotBeDeleted is returned when attempting to delete a built-in permission.
var ErrBuiltInPermissionCannotBeDeleted = errors.New("built-in permission cannot be deleted")

var _ port.PermissionUsecase = (*PermissionService)(nil)

// PermissionService implements the PermissionUsecase interface.
type PermissionService struct {
	permissionPersistencePort port.PermissionPersistencePort
	logger                    *slog.Logger
}

// NewPermissionService creates a new instance of PermissionService.
func NewPermissionService(
	permissionPersistencePort port.PermissionPersistencePort,
	logger *slog.Logger,
) *PermissionService {
	return &PermissionService{
		permissionPersistencePort: permissionPersistencePort,
		logger:                    logger,
	}
}

// GetPermission implements [port.PermissionUsecase].
func (s *PermissionService) GetPermission(
	ctx context.Context,
	uid uuid.UUID,
) (*model.Permission, error) {
	permission, err := s.permissionPersistencePort.GetPermission(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to get permission from persistence: %w", err)
	}

	return permission, nil
}

// GetPermissionByName implements [port.PermissionUsecase].
func (s *PermissionService) GetPermissionByName(
	ctx context.Context,
	name string,
) (*model.Permission, error) {
	permission, err := s.permissionPersistencePort.GetPermissionByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get permission by name from persistence: %w", err)
	}

	return permission, nil
}

// ListPermissions implements [port.PermissionUsecase].
func (s *PermissionService) ListPermissions(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*model.Permission], error) {
	resp, err := s.permissionPersistencePort.ListPermissions(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list permissions from persistence: %w", err)
	}

	return resp, nil
}

// SavePermission implements [port.PermissionUsecase].
func (s *PermissionService) SavePermission(
	ctx context.Context,
	permission *model.Permission,
) error {
	_, err := s.permissionPersistencePort.PutPermission(ctx, permission)
	if err != nil {
		return fmt.Errorf("failed to save permission to persistence: %w", err)
	}

	return nil
}

// DeletePermission implements [port.PermissionUsecase].
func (s *PermissionService) DeletePermission(
	ctx context.Context,
	uid uuid.UUID,
) error {
	permission, err := s.permissionPersistencePort.GetPermission(ctx, uid)
	if err != nil {
		return fmt.Errorf("failed to get permission from persistence: %w", err)
	}

	if permission.Spec.IsBuiltIn {
		return ErrBuiltInPermissionCannotBeDeleted
	}

	err = s.permissionPersistencePort.DeletePermission(ctx, uid)
	if err != nil {
		return fmt.Errorf("failed to delete permission from persistence: %w", err)
	}

	return nil
}
