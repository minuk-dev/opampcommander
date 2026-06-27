package usecase

import (
	"context"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// NamespaceManageUsecase is a use case that handles namespace management operations.
type NamespaceManageUsecase interface {
	GetNamespace(ctx context.Context, name string,
		options *port.GetOptions) (*v1.Namespace, error)
	ListNamespaces(ctx context.Context,
		options *port.ListOptions) (*v1.ListResponse[v1.Namespace], error)
	CreateNamespace(ctx context.Context,
		namespace *v1.Namespace) (*v1.Namespace, error)
	UpdateNamespace(ctx context.Context, name string,
		namespace *v1.Namespace) (*v1.Namespace, error)
	DeleteNamespace(ctx context.Context, name string) error
}
