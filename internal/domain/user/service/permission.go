//nolint:dupl // Service pattern - similar structure is intentional.
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

// ErrBuiltInPermissionCannotBeDeleted is returned when attempting to delete a built-in permission.
var ErrBuiltInPermissionCannotBeDeleted = errors.New("built-in permission cannot be deleted")

var _ userport.PermissionUsecase = (*PermissionService)(nil)

// PermissionService implements the PermissionUsecase interface.
type PermissionService struct {
	permissionPersistencePort userport.PermissionPersistencePort
	logger                    *slog.Logger
}

// NewPermissionService creates a new instance of PermissionService.
func NewPermissionService(
	permissionPersistencePort userport.PermissionPersistencePort,
	logger *slog.Logger,
) *PermissionService {
	return &PermissionService{
		permissionPersistencePort: permissionPersistencePort,
		logger:                    logger,
	}
}

// GetPermission implements [userport.PermissionUsecase].
func (s *PermissionService) GetPermission(
	ctx context.Context,
	uid uuid.UUID,
) (*usermodel.Permission, error) {
	permission, err := s.permissionPersistencePort.GetPermission(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to get permission from persistence: %w", err)
	}

	return permission, nil
}

// GetPermissionByName implements [userport.PermissionUsecase].
func (s *PermissionService) GetPermissionByName(
	ctx context.Context,
	name string,
) (*usermodel.Permission, error) {
	permission, err := s.permissionPersistencePort.GetPermissionByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get permission by name from persistence: %w", err)
	}

	return permission, nil
}

// ListPermissions implements [userport.PermissionUsecase].
func (s *PermissionService) ListPermissions(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*usermodel.Permission], error) {
	resp, err := s.permissionPersistencePort.ListPermissions(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list permissions from persistence: %w", err)
	}

	return resp, nil
}

// SavePermission implements [userport.PermissionUsecase].
func (s *PermissionService) SavePermission(
	ctx context.Context,
	permission *usermodel.Permission,
) error {
	_, err := s.permissionPersistencePort.PutPermission(ctx, permission)
	if err != nil {
		return fmt.Errorf("failed to save permission to persistence: %w", err)
	}

	return nil
}

// DeletePermission implements [userport.PermissionUsecase].
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
