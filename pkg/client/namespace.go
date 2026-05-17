package client

import (
	"context"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

const (
	// ListNamespaceURL is the path to list all namespaces.
	ListNamespaceURL = "/api/v1/namespaces"
	// GetNamespaceURL is the path to get a namespace by name.
	GetNamespaceURL = "/api/v1/namespaces/{id}"
	// CreateNamespaceURL is the path to create a new namespace.
	CreateNamespaceURL = "/api/v1/namespaces"
	// UpdateNamespaceURL is the path to update an existing namespace.
	UpdateNamespaceURL = "/api/v1/namespaces/{id}"
	// DeleteNamespaceURL is the path to delete a namespace.
	DeleteNamespaceURL = "/api/v1/namespaces/{id}"
)

// NamespaceService provides methods to interact with namespaces.
type NamespaceService struct {
	service *service
}

// NewNamespaceService creates a new NamespaceService.
func NewNamespaceService(service *service) *NamespaceService {
	return &NamespaceService{
		service: service,
	}
}

// GetNamespace retrieves a namespace by its name.
func (s *NamespaceService) GetNamespace(
	ctx context.Context,
	name string,
	opts ...GetOption,
) (*v1.Namespace, error) {
	return getResource[v1.Namespace](
		ctx, s.service, GetNamespaceURL, name, opts...,
	)
}

// NamespaceListResponse represents a list of namespaces with metadata.
type NamespaceListResponse = v1.ListResponse[v1.Namespace]

// ListNamespaces lists all namespaces.
func (s *NamespaceService) ListNamespaces(
	ctx context.Context,
	opts ...ListOption,
) (*NamespaceListResponse, error) {
	var listSettings ListSettings

	for _, opt := range opts {
		opt.Apply(&listSettings)
	}

	return listResources[v1.Namespace](
		ctx,
		s.service,
		ListNamespaceURL,
		ListSettings{
			limit:          listSettings.limit,
			continueToken:  listSettings.continueToken,
			includeDeleted: listSettings.includeDeleted,
		},
	)
}

// CreateNamespace creates a new namespace.
func (s *NamespaceService) CreateNamespace(
	ctx context.Context,
	createRequest *v1.Namespace,
) (*v1.Namespace, error) {
	return createResource[v1.Namespace, v1.Namespace](
		ctx, s.service, CreateNamespaceURL, createRequest,
	)
}

// UpdateNamespace updates an existing namespace.
func (s *NamespaceService) UpdateNamespace(
	ctx context.Context,
	name string,
	updateRequest *v1.Namespace,
) (*v1.Namespace, error) {
	return updateResource[v1.Namespace](
		ctx, s.service, UpdateNamespaceURL, name, updateRequest,
	)
}

// DeleteNamespace deletes a namespace by name.
func (s *NamespaceService) DeleteNamespace(
	ctx context.Context,
	name string,
) error {
	return deleteResource(ctx, s.service, DeleteNamespaceURL, name)
}
