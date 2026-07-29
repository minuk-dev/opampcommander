package usecase

import (
	"context"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// RemoteConfigSchemaManageUsecase manages remote config schemas: the collector build
// (distribution + version) and component catalog a remote config is validated
// against. It backs the /api/v1/remoteconfigschemas controller.
type RemoteConfigSchemaManageUsecase interface {
	// GetRemoteConfigSchema returns the named schema in namespace, or
	// model.ErrResourceNotExist if absent.
	GetRemoteConfigSchema(ctx context.Context, namespace string,
		name string, options *port.GetOptions) (*v1.RemoteConfigSchema, error)
	// ListRemoteConfigSchemas returns a paged list of schemas in namespace.
	ListRemoteConfigSchemas(ctx context.Context, namespace string,
		options *port.ListOptions) (*v1.ListResponse[v1.RemoteConfigSchema], error)
	// CreateRemoteConfigSchema persists a new schema, returning
	// model.ErrResourceAlreadyExist on a duplicate.
	CreateRemoteConfigSchema(ctx context.Context,
		schema *v1.RemoteConfigSchema) (*v1.RemoteConfigSchema, error)
	// UpdateRemoteConfigSchema replaces the named schema; optimistic-concurrency
	// controlled (model.ErrConflict on a stale write).
	UpdateRemoteConfigSchema(ctx context.Context, namespace string, name string,
		schema *v1.RemoteConfigSchema) (*v1.RemoteConfigSchema, error)
	// DeleteRemoteConfigSchema removes the named schema.
	DeleteRemoteConfigSchema(ctx context.Context, namespace string, name string) error
}
