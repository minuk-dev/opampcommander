package userservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	usermodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user"
	userport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user/port"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

// ErrBuiltInRoleCannotBeDeleted is returned when attempting to delete a built-in role.
var ErrBuiltInRoleCannotBeDeleted = errors.New("built-in role cannot be deleted")

var _ userport.RoleUsecase = (*RoleService)(nil)

// RoleService implements the RoleUsecase interface, owning the role lifecycle
// rules (user-created roles are never built-in, IsBuiltIn is immutable on update,
// and update timestamps are stamped).
type RoleService struct {
	rolePersistencePort userport.RolePersistencePort
	clock               clock.Clock
	logger              *slog.Logger
}

// NewRoleService creates a new instance of RoleService.
func NewRoleService(
	rolePersistencePort userport.RolePersistencePort,
	logger *slog.Logger,
) *RoleService {
	return &RoleService{
		rolePersistencePort: rolePersistencePort,
		clock:               clock.NewRealClock(),
		logger:              logger,
	}
}

// SetClock overrides the clock used for lifecycle timestamps. Intended for tests.
func (s *RoleService) SetClock(c clock.Clock) {
	s.clock = c
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

// CreateRole implements [userport.RoleUsecase]. User-created roles are forced to be
// non-built-in, and their creation/update timestamps are stamped from the clock.
func (s *RoleService) CreateRole(
	ctx context.Context,
	role *usermodel.Role,
) (*usermodel.Role, error) {
	role.Spec.IsBuiltIn = false

	now := s.clock.Now()
	role.Metadata.CreatedAt = now
	role.Metadata.UpdatedAt = now

	if role.Metadata.UID == uuid.Nil {
		role.Metadata.UID = uuid.New()
	}

	_, err := s.rolePersistencePort.PutRole(ctx, role)
	if err != nil {
		return nil, fmt.Errorf("failed to create role in persistence: %w", err)
	}

	return role, nil
}

// UpdateRole implements [userport.RoleUsecase]. It applies the mutable spec fields
// onto the stored role, keeping IsBuiltIn immutable, and stamps the update time.
func (s *RoleService) UpdateRole(
	ctx context.Context,
	uid uuid.UUID,
	role *usermodel.Role,
) (*usermodel.Role, error) {
	existing, err := s.rolePersistencePort.GetRole(ctx, uid, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get role for update: %w", err)
	}

	existing.Spec.DisplayName = role.Spec.DisplayName
	existing.Spec.Description = role.Spec.Description
	existing.Spec.Permissions = role.Spec.Permissions
	// IsBuiltIn is immutable — not updated from the request.
	existing.Metadata.UpdatedAt = s.clock.Now()

	_, err = s.rolePersistencePort.PutRole(ctx, existing)
	if err != nil {
		return nil, fmt.Errorf("failed to update role in persistence: %w", err)
	}

	return existing, nil
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
