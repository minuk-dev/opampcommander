package agentservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	domainport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var _ agentport.EndpointUsecase = (*EndpointService)(nil)

// EndpointService provides operations for managing endpoints, including the
// creation/update lifecycle rules (identity validation, uniqueness, stamping, and
// immutable-field preservation).
type EndpointService struct {
	persistence agentport.EndpointPersistencePort
	clock       clock.Clock
}

// NewEndpointService creates a new EndpointService.
func NewEndpointService(persistence agentport.EndpointPersistencePort) *EndpointService {
	return &EndpointService{
		persistence: persistence,
		clock:       clock.NewRealClock(),
	}
}

// SetClock overrides the clock used for lifecycle timestamps. Intended for tests.
func (s *EndpointService) SetClock(c clock.Clock) {
	s.clock = c
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
	namespace string,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Endpoint], error) {
	resourceResp, err := s.persistence.ListEndpoints(ctx, namespace, options)
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

// CreateEndpoint implements [agentport.EndpointUsecase].
func (s *EndpointService) CreateEndpoint(
	ctx context.Context,
	endpoint *agentmodel.Endpoint,
	actor string,
) (*agentmodel.Endpoint, error) {
	if endpoint.Metadata.Name == "" {
		return nil, fmt.Errorf("%w: endpoint name must not be empty", domainport.ErrInvalidArgument)
	}

	// Reject creating over an existing endpoint instead of silently upserting it.
	_, err := s.persistence.GetEndpoint(ctx, endpoint.Metadata.Namespace, endpoint.Metadata.Name, nil)
	switch {
	case err == nil:
		return nil, fmt.Errorf("%w: endpoint %q in namespace %q",
			domainport.ErrResourceAlreadyExist, endpoint.Metadata.Name, endpoint.Metadata.Namespace)
	case !errors.Is(err, domainport.ErrResourceNotExist):
		return nil, fmt.Errorf("check existing endpoint: %w", err)
	}

	endpoint.MarkAsCreated(s.clock.Now(), actor)

	created, err := s.persistence.PutEndpoint(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create endpoint: %w", err)
	}

	return created, nil
}

// UpdateEndpoint implements [agentport.EndpointUsecase].
func (s *EndpointService) UpdateEndpoint(
	ctx context.Context,
	namespace string,
	name string,
	endpoint *agentmodel.Endpoint,
) (*agentmodel.Endpoint, error) {
	existing, err := s.persistence.GetEndpoint(ctx, namespace, name, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint for update: %w", err)
	}

	existing.ApplyUpdate(endpoint)

	updated, err := s.persistence.PutEndpoint(ctx, existing)
	if err != nil {
		return nil, fmt.Errorf("failed to update endpoint: %w", err)
	}

	return updated, nil
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
