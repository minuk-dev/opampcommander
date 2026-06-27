// Package endpoint provides the service for managing endpoints.
package endpoint

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/helper"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/security"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var _ port.EndpointManageUsecase = (*Service)(nil)

// Service is a service for managing endpoints. It maps between the HTTP DTOs and
// the domain, resolves the acting user, and delegates all lifecycle rules
// (identity validation, uniqueness, stamping, immutable-field preservation) to the
// domain EndpointUsecase.
type Service struct {
	endpointUsecase agentport.EndpointUsecase
	mapper          *helper.Mapper
	clock           clock.Clock
	logger          *slog.Logger
}

// NewEndpointService creates a new endpoint Service.
func NewEndpointService(
	endpointUsecase agentport.EndpointUsecase,
	logger *slog.Logger,
) *Service {
	realClock := clock.NewRealClock()

	return &Service{
		endpointUsecase: endpointUsecase,
		mapper:          helper.NewMapper(realClock, 0),
		clock:           realClock,
		logger:          logger,
	}
}

// GetEndpoint implements [port.EndpointManageUsecase].
func (s *Service) GetEndpoint(
	ctx context.Context,
	namespace string,
	name string,
	options *port.GetOptions,
) (*v1.Endpoint, error) {
	endpoint, err := s.endpointUsecase.GetEndpoint(ctx, namespace, name, options.ToDomain())
	if err != nil {
		return nil, fmt.Errorf("get endpoint: %w", err)
	}

	return s.mapper.MapEndpointToAPI(endpoint), nil
}

// ListEndpoints implements [port.EndpointManageUsecase].
func (s *Service) ListEndpoints(
	ctx context.Context,
	namespace string,
	options *port.ListOptions,
) (*v1.ListResponse[v1.Endpoint], error) {
	endpoints, err := s.endpointUsecase.ListEndpoints(ctx, namespace, options.ToDomain())
	if err != nil {
		return nil, fmt.Errorf("list endpoints: %w", err)
	}

	return &v1.ListResponse[v1.Endpoint]{
		Kind:       v1.EndpointKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.ListMeta{
			Continue:           endpoints.Continue,
			RemainingItemCount: endpoints.RemainingItemCount,
		},
		Items: lo.Map(
			endpoints.Items,
			func(item *agentmodel.Endpoint, _ int) v1.Endpoint {
				return *s.mapper.MapEndpointToAPI(item)
			},
		),
	}, nil
}

// CreateEndpoint implements [port.EndpointManageUsecase].
func (s *Service) CreateEndpoint(
	ctx context.Context,
	apiModel *v1.Endpoint,
) (*v1.Endpoint, error) {
	domainModel := s.mapper.MapAPIToEndpoint(apiModel)

	saved, err := s.endpointUsecase.CreateEndpoint(ctx, domainModel, s.actor(ctx))
	if err != nil {
		return nil, fmt.Errorf("create endpoint: %w", err)
	}

	return s.mapper.MapEndpointToAPI(saved), nil
}

// UpdateEndpoint implements [port.EndpointManageUsecase].
func (s *Service) UpdateEndpoint(
	ctx context.Context,
	namespace string,
	name string,
	apiModel *v1.Endpoint,
) (*v1.Endpoint, error) {
	domainModel := s.mapper.MapAPIToEndpoint(apiModel)

	updated, err := s.endpointUsecase.UpdateEndpoint(ctx, namespace, name, domainModel)
	if err != nil {
		return nil, fmt.Errorf("update endpoint: %w", err)
	}

	return s.mapper.MapEndpointToAPI(updated), nil
}

// DeleteEndpoint implements [port.EndpointManageUsecase].
func (s *Service) DeleteEndpoint(
	ctx context.Context,
	namespace string,
	name string,
) error {
	err := s.endpointUsecase.DeleteEndpoint(
		ctx, namespace, name, s.clock.Now(), s.actor(ctx),
	)
	if err != nil {
		return fmt.Errorf("delete endpoint: %w", err)
	}

	return nil
}

// actor resolves the acting user from the request context, falling back to an
// anonymous identity (and logging) when none is present.
func (s *Service) actor(ctx context.Context) string {
	user, err := security.GetUser(ctx)
	if err != nil {
		s.logger.Warn(
			"failed to get user from context",
			slog.String("error", err.Error()),
		)

		user = security.NewAnonymousUser()
	}

	return user.String()
}
