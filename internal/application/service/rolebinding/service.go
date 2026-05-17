// Package rolebinding provides application services for RoleBinding management.
package rolebinding

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/internal/application/helper"
	applicationport "github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	usermodel "github.com/minuk-dev/opampcommander/internal/domain/user/model"
	userport "github.com/minuk-dev/opampcommander/internal/domain/user/port"
)

var _ applicationport.RoleBindingManageUsecase = (*Service)(nil)

// Service implements the RoleBindingManageUsecase interface.
type Service struct {
	roleBindingUsecase userport.RoleBindingUsecase
	roleUsecase        userport.RoleUsecase
	rbacUsecase        userport.RBACUsecase
	mapper             *helper.Mapper
	logger             *slog.Logger
}

// New creates a new instance of the Service struct.
func New(
	roleBindingUsecase userport.RoleBindingUsecase,
	roleUsecase userport.RoleUsecase,
	rbacUsecase userport.RBACUsecase,
	logger *slog.Logger,
) *Service {
	return &Service{
		roleBindingUsecase: roleBindingUsecase,
		roleUsecase:        roleUsecase,
		rbacUsecase:        rbacUsecase,
		mapper:             helper.NewMapper(),
		logger:             logger,
	}
}

// GetRoleBinding implements [applicationport.RoleBindingManageUsecase].
func (s *Service) GetRoleBinding(
	ctx context.Context,
	namespace, name string,
	options *model.GetOptions,
) (*v1.RoleBinding, error) {
	rb, err := s.roleBindingUsecase.GetRoleBinding(ctx, namespace, name, options)
	if err != nil {
		return nil, fmt.Errorf("get role binding: %w", err)
	}

	return s.mapper.MapRoleBindingToAPI(rb), nil
}

// ListRoleBindings implements [applicationport.RoleBindingManageUsecase].
func (s *Service) ListRoleBindings(
	ctx context.Context,
	options *model.ListOptions,
) (*v1.ListResponse[v1.RoleBinding], error) {
	domainResp, err := s.roleBindingUsecase.ListRoleBindings(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("list role bindings: %w", err)
	}

	return &v1.ListResponse[v1.RoleBinding]{
		Kind:       v1.RoleBindingKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.ListMeta{
			Continue:           domainResp.Continue,
			RemainingItemCount: domainResp.RemainingItemCount,
		},
		Items: lo.Map(domainResp.Items, func(rb *usermodel.RoleBinding, _ int) v1.RoleBinding {
			return *s.mapper.MapRoleBindingToAPI(rb)
		}),
	}, nil
}

// CreateRoleBinding implements [applicationport.RoleBindingManageUsecase].
func (s *Service) CreateRoleBinding(
	ctx context.Context,
	apiRB *v1.RoleBinding,
) (*v1.RoleBinding, error) {
	// Validate the role exists before persisting the binding.
	_, err := s.roleUsecase.GetRoleByName(ctx, apiRB.Spec.RoleRef.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve role %q: %w", apiRB.Spec.RoleRef.Name, err)
	}

	created, err := s.roleBindingUsecase.CreateRoleBinding(ctx, s.mapper.MapAPIToRoleBinding(apiRB))
	if err != nil {
		return nil, fmt.Errorf("create role binding: %w", err)
	}

	s.syncPolicies(ctx, "create", created.Metadata.Namespace, created.Metadata.Name)

	return s.mapper.MapRoleBindingToAPI(created), nil
}

// UpdateRoleBinding implements [applicationport.RoleBindingManageUsecase].
func (s *Service) UpdateRoleBinding(
	ctx context.Context,
	namespace, name string,
	apiRB *v1.RoleBinding,
) (*v1.RoleBinding, error) {
	// Validate the role exists before persisting the binding.
	_, err := s.roleUsecase.GetRoleByName(ctx, apiRB.Spec.RoleRef.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve role %q: %w", apiRB.Spec.RoleRef.Name, err)
	}

	updated, err := s.roleBindingUsecase.UpdateRoleBinding(ctx, namespace, name, s.mapper.MapAPIToRoleBinding(apiRB))
	if err != nil {
		return nil, fmt.Errorf("update role binding: %w", err)
	}

	s.syncPolicies(ctx, "update", namespace, name)

	return s.mapper.MapRoleBindingToAPI(updated), nil
}

// DeleteRoleBinding implements [applicationport.RoleBindingManageUsecase].
func (s *Service) DeleteRoleBinding(
	ctx context.Context,
	namespace, name string,
) error {
	err := s.roleBindingUsecase.DeleteRoleBinding(ctx, namespace, name)
	if err != nil {
		return fmt.Errorf("delete role binding: %w", err)
	}

	s.syncPolicies(ctx, "delete", namespace, name)

	return nil
}

// syncPolicies re-runs the Casbin policy sync after a binding mutation so the change
// takes effect without requiring a server restart. Best-effort: failures are logged
// but do not fail the API call (the binding is already persisted).
func (s *Service) syncPolicies(ctx context.Context, op, namespace, name string) {
	err := s.rbacUsecase.SyncPolicies(ctx)
	if err != nil {
		s.logger.WarnContext(ctx, "failed to sync RBAC policies after role binding mutation",
			slog.String("op", op),
			slog.String("namespace", namespace),
			slog.String("name", name),
			slog.Any("error", err),
		)
	}
}
