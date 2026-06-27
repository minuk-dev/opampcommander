package usecase

import (
	"context"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// NamespaceManageUsecase manages namespaces: the tenancy boundary every
// other resource is scoped by. It backs the /api/v1/namespaces controller.
type NamespaceManageUsecase interface {
	// GetNamespace returns the named namespace, or model.ErrResourceNotExist if
	// absent.
	GetNamespace(ctx context.Context, name string,
		options *port.GetOptions) (*v1.Namespace, error)
	// ListNamespaces returns a paged list of namespaces.
	ListNamespaces(ctx context.Context,
		options *port.ListOptions) (*v1.ListResponse[v1.Namespace], error)
	// CreateNamespace persists a new namespace, returning
	// ErrNamespaceAlreadyExists on a duplicate.
	CreateNamespace(ctx context.Context,
		namespace *v1.Namespace) (*v1.Namespace, error)
	// UpdateNamespace replaces the named namespace's spec.
	UpdateNamespace(ctx context.Context, name string,
		namespace *v1.Namespace) (*v1.Namespace, error)
	// DeleteNamespace removes the named namespace. The built-in default
	// namespace is protected and yields ErrDefaultNamespaceUndeletable.
	DeleteNamespace(ctx context.Context, name string) error
}
