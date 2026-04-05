package agentservice

import (
	"context"
	"fmt"
	"time"

	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/internal/domain/agent/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

var _ agentport.NamespaceUsecase = (*NamespaceService)(nil)

// NamespaceService provides operations for managing namespaces.
type NamespaceService struct {
	persistence agentport.NamespacePersistencePort
}

// NewNamespaceService creates a new NamespaceService.
func NewNamespaceService(
	persistence agentport.NamespacePersistencePort,
) *NamespaceService {
	return &NamespaceService{
		persistence: persistence,
	}
}

// GetNamespace implements [agentport.NamespaceUsecase].
func (s *NamespaceService) GetNamespace(
	ctx context.Context,
	name string,
) (*agentmodel.Namespace, error) {
	namespace, err := s.persistence.GetNamespace(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace: %w", err)
	}

	return namespace, nil
}

// ListNamespaces implements [agentport.NamespaceUsecase].
func (s *NamespaceService) ListNamespaces(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Namespace], error) {
	resp, err := s.persistence.ListNamespaces(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	return resp, nil
}

// SaveNamespace implements [agentport.NamespaceUsecase].
func (s *NamespaceService) SaveNamespace(
	ctx context.Context,
	namespace *agentmodel.Namespace,
) (*agentmodel.Namespace, error) {
	saved, err := s.persistence.PutNamespace(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to save namespace: %w", err)
	}

	return saved, nil
}

// DeleteNamespace implements [agentport.NamespaceUsecase].
func (s *NamespaceService) DeleteNamespace(
	ctx context.Context,
	name string,
	deletedAt time.Time,
	deletedBy string,
) error {
	namespace, err := s.persistence.GetNamespace(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get namespace for deletion: %w", err)
	}

	namespace.MarkAsDeleted(deletedAt, deletedBy)

	_, err = s.persistence.PutNamespace(ctx, namespace)
	if err != nil {
		return fmt.Errorf("failed to mark namespace as deleted: %w", err)
	}

	return nil
}
