package usecase

import (
	"context"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// RoleBindingManageUsecase manages RBAC role bindings: assignments of a role
// to subjects within a namespace. It backs the /api/v1/rolebindings
// controller.
type RoleBindingManageUsecase interface {
	// GetRoleBinding returns the named binding in namespace, or
	// model.ErrResourceNotExist if absent.
	GetRoleBinding(ctx context.Context, namespace, name string,
		options *port.GetOptions) (*v1.RoleBinding, error)
	// ListRoleBindings returns a paged list of bindings across namespaces.
	ListRoleBindings(ctx context.Context, options *port.ListOptions) (*v1.ListResponse[v1.RoleBinding], error)
	// CreateRoleBinding persists a new binding.
	CreateRoleBinding(ctx context.Context, rb *v1.RoleBinding) (*v1.RoleBinding, error)
	// UpdateRoleBinding replaces the named binding.
	UpdateRoleBinding(ctx context.Context, namespace, name string, rb *v1.RoleBinding) (*v1.RoleBinding, error)
	// DeleteRoleBinding removes the named binding.
	DeleteRoleBinding(ctx context.Context, namespace, name string) error
}
