package userservice

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	usermodel "github.com/minuk-dev/opampcommander/internal/domain/user/model"
	userport "github.com/minuk-dev/opampcommander/internal/domain/user/port"
)

var _ userport.RoleBindingUsecase = (*RoleBindingService)(nil)

// RoleBindingService implements the RoleBindingUsecase interface.
type RoleBindingService struct {
	roleBindingPersistencePort userport.RoleBindingPersistencePort
	logger                     *slog.Logger
}

// NewRoleBindingService creates a new instance of RoleBindingService.
func NewRoleBindingService(
	roleBindingPersistencePort userport.RoleBindingPersistencePort,
	logger *slog.Logger,
) *RoleBindingService {
	return &RoleBindingService{
		roleBindingPersistencePort: roleBindingPersistencePort,
		logger:                     logger,
	}
}

// GetRoleBinding implements [userport.RoleBindingUsecase].
func (s *RoleBindingService) GetRoleBinding(
	ctx context.Context,
	namespace, name string,
) (*usermodel.RoleBinding, error) {
	rb, err := s.roleBindingPersistencePort.GetRoleBinding(ctx, namespace, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get role binding from persistence: %w", err)
	}

	return rb, nil
}

// ListRoleBindings implements [userport.RoleBindingUsecase].
func (s *RoleBindingService) ListRoleBindings(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*usermodel.RoleBinding], error) {
	resp, err := s.roleBindingPersistencePort.ListRoleBindings(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list role bindings from persistence: %w", err)
	}

	return resp, nil
}

// CreateRoleBinding implements [userport.RoleBindingUsecase].
func (s *RoleBindingService) CreateRoleBinding(
	ctx context.Context,
	roleBinding *usermodel.RoleBinding,
) (*usermodel.RoleBinding, error) {
	created, err := s.roleBindingPersistencePort.PutRoleBinding(ctx, roleBinding)
	if err != nil {
		return nil, fmt.Errorf("failed to create role binding in persistence: %w", err)
	}

	return created, nil
}

// UpdateRoleBinding implements [userport.RoleBindingUsecase].
func (s *RoleBindingService) UpdateRoleBinding(
	ctx context.Context,
	namespace, name string,
	roleBinding *usermodel.RoleBinding,
) (*usermodel.RoleBinding, error) {
	roleBinding.Metadata.Namespace = namespace
	roleBinding.Metadata.Name = name
	roleBinding.SetUpdatedAt(time.Now())

	updated, err := s.roleBindingPersistencePort.PutRoleBinding(ctx, roleBinding)
	if err != nil {
		return nil, fmt.Errorf("failed to update role binding in persistence: %w", err)
	}

	return updated, nil
}

// DeleteRoleBinding implements [userport.RoleBindingUsecase].
func (s *RoleBindingService) DeleteRoleBinding(
	ctx context.Context,
	namespace, name string,
) error {
	err := s.roleBindingPersistencePort.DeleteRoleBinding(ctx, namespace, name)
	if err != nil {
		return fmt.Errorf("failed to delete role binding: %w", err)
	}

	return nil
}
