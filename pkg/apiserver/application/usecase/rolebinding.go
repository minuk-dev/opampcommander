package usecase

import (
	"context"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// RoleBindingManageUsecase is a use case that handles RoleBinding management operations.
type RoleBindingManageUsecase interface {
	GetRoleBinding(ctx context.Context, namespace, name string,
		options *port.GetOptions) (*v1.RoleBinding, error)
	ListRoleBindings(ctx context.Context, options *port.ListOptions) (*v1.ListResponse[v1.RoleBinding], error)
	CreateRoleBinding(ctx context.Context, rb *v1.RoleBinding) (*v1.RoleBinding, error)
	UpdateRoleBinding(ctx context.Context, namespace, name string, rb *v1.RoleBinding) (*v1.RoleBinding, error)
	DeleteRoleBinding(ctx context.Context, namespace, name string) error
}
