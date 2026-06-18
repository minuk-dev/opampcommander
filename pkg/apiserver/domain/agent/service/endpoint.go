//nolint:dupl // namespaced CRUD domain services intentionally share this shape.
package agentservice

import (
	"context"
	"fmt"
	"time"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var _ agentport.EndpointUsecase = (*EndpointService)(nil)

// EndpointService provides operations for managing endpoints.
type EndpointService struct {
	persistence agentport.EndpointPersistencePort
}

// NewEndpointService creates a new EndpointService.
func NewEndpointService(persistence agentport.EndpointPersistencePort) *EndpointService {
	return &EndpointService{
		persistence: persistence,
	}
}

// GetEndpoint implements [agentport.EndpointUsecase].
func (s *EndpointService) GetEndpoint(
	ctx context.Context,
	namespace string,
	name string,
	options *model.GetOptions,
) (*agentmodel.Endpoint, error) {
	resource, err := s.persistence.GetEndpoint(ctx, namespace, name, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint: %w", err)
	}

	return resource, nil
}

// ListEndpoints implements [agentport.EndpointUsecase].
func (s *EndpointService) ListEndpoints(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Endpoint], error) {
	resourceResp, err := s.persistence.ListEndpoints(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list endpoints: %w", err)
	}

	return &model.ListResponse[*agentmodel.Endpoint]{
		Items:              resourceResp.Items,
		RemainingItemCount: resourceResp.RemainingItemCount,
		Continue:           resourceResp.Continue,
	}, nil
}

// SaveEndpoint implements [agentport.EndpointUsecase].
func (s *EndpointService) SaveEndpoint(
	ctx context.Context,
	endpoint *agentmodel.Endpoint,
) (*agentmodel.Endpoint, error) {
	resource, err := s.persistence.PutEndpoint(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to save endpoint: %w", err)
	}

	return resource, nil
}

// DeleteEndpoint implements [agentport.EndpointUsecase].
func (s *EndpointService) DeleteEndpoint(
	ctx context.Context,
	namespace string,
	name string,
	deletedAt time.Time,
	deletedBy string,
) error {
	resource, err := s.persistence.GetEndpoint(ctx, namespace, name, nil)
	if err != nil {
		return fmt.Errorf("failed to get endpoint for deletion: %w", err)
	}

	resource.MarkDeleted(deletedAt, deletedBy)

	_, err = s.persistence.PutEndpoint(ctx, resource)
	if err != nil {
		return fmt.Errorf("failed to mark endpoint as deleted: %w", err)
	}

	return nil
}
