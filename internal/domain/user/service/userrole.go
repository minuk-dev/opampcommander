package userservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	usermodel "github.com/minuk-dev/opampcommander/internal/domain/user/model"
	userport "github.com/minuk-dev/opampcommander/internal/domain/user/port"
)

var _ userport.UserRoleUsecase = (*UserRoleService)(nil)

var (
	// ErrRoleAlreadyAssigned is returned when a role is already assigned to a user.
	ErrRoleAlreadyAssigned = errors.New("role is already assigned to user")
)

// UserRoleService implements the UserRoleUsecase interface.
type UserRoleService struct {
	userRolePersistencePort userport.UserRolePersistencePort
	logger                  *slog.Logger
}

// NewUserRoleService creates a new instance of UserRoleService.
func NewUserRoleService(
	userRolePersistencePort userport.UserRolePersistencePort,
	logger *slog.Logger,
) *UserRoleService {
	return &UserRoleService{
		userRolePersistencePort: userRolePersistencePort,
		logger:                  logger,
	}
}

// AssignRole implements [userport.UserRoleUsecase].
func (s *UserRoleService) AssignRole(
	ctx context.Context,
	userID, roleID, assignedBy uuid.UUID,
) error {
	// Check if the role is already assigned
	existing, err := s.userRolePersistencePort.GetUserRoleByUserAndRole(ctx, userID, roleID)
	if err != nil && !errors.Is(err, port.ErrResourceNotExist) {
		return fmt.Errorf("failed to check existing role assignment: %w", err)
	}

	if existing != nil {
		return ErrRoleAlreadyAssigned
	}

	userRole := usermodel.NewUserRole(userID, roleID, assignedBy)

	_, err = s.userRolePersistencePort.PutUserRole(ctx, userRole)
	if err != nil {
		return fmt.Errorf("failed to assign role to persistence: %w", err)
	}

	return nil
}

// UnassignRole implements [userport.UserRoleUsecase].
func (s *UserRoleService) UnassignRole(
	ctx context.Context,
	userID, roleID uuid.UUID,
) error {
	existing, err := s.userRolePersistencePort.GetUserRoleByUserAndRole(ctx, userID, roleID)
	if err != nil {
		return fmt.Errorf("failed to find user role assignment: %w", err)
	}

	err = s.userRolePersistencePort.DeleteUserRole(ctx, existing.Metadata.UID)
	if err != nil {
		return fmt.Errorf("failed to delete user role from persistence: %w", err)
	}

	return nil
}

// GetUserRoles implements [userport.UserRoleUsecase].
func (s *UserRoleService) GetUserRoles(
	ctx context.Context,
	userID uuid.UUID,
) ([]*usermodel.Role, error) {
	roles, err := s.userRolePersistencePort.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles from persistence: %w", err)
	}

	return roles, nil
}

// GetRoleUsers implements [userport.UserRoleUsecase].
func (s *UserRoleService) GetRoleUsers(
	ctx context.Context,
	roleID uuid.UUID,
) ([]*usermodel.User, error) {
	users, err := s.userRolePersistencePort.GetRoleUsers(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role users from persistence: %w", err)
	}

	return users, nil
}

// ListUserRoles implements [userport.UserRoleUsecase].
func (s *UserRoleService) ListUserRoles(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*usermodel.UserRole], error) {
	resp, err := s.userRolePersistencePort.ListUserRoles(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list user roles from persistence: %w", err)
	}

	return resp, nil
}
