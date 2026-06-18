// Package endpoint provides the service for managing endpoints.
package endpoint

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/helper"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/endpoint/filter"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	domainport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/security"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var _ port.EndpointManageUsecase = (*Service)(nil)

// Service is a service for managing endpoints.
type Service struct {
	endpointUsecase agentport.EndpointUsecase
	mapper          *helper.Mapper
	sanityFilter    *filter.Sanity
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
		sanityFilter:    filter.NewSanity(),
		clock:           realClock,
		logger:          logger,
	}
}

// GetEndpoint implements [port.EndpointManageUsecase].
func (s *Service) GetEndpoint(
	ctx context.Context,
	namespace string,
	name string,
	options *model.GetOptions,
) (*v1.Endpoint, error) {
	endpoint, err := s.endpointUsecase.GetEndpoint(ctx, namespace, name, options)
	if err != nil {
		return nil, fmt.Errorf("get endpoint: %w", err)
	}

	return s.mapper.MapEndpointToAPI(endpoint), nil
}

// ListEndpoints implements [port.EndpointManageUsecase].
func (s *Service) ListEndpoints(
	ctx context.Context,
	namespace string,
	options *model.ListOptions,
) (*v1.ListResponse[v1.Endpoint], error) {
	endpoints, err := s.endpointUsecase.ListEndpoints(ctx, namespace, options)
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

	if domainModel.Metadata.Name == "" {
		return nil, fmt.Errorf("%w: endpoint name must not be empty", domainport.ErrInvalidArgument)
	}

	// Reject creating over an existing endpoint instead of silently upserting it.
	_, err := s.endpointUsecase.GetEndpoint(ctx, domainModel.Metadata.Namespace, domainModel.Metadata.Name, nil)
	switch {
	case err == nil:
		return nil, fmt.Errorf("%w: endpoint %q in namespace %q",
			domainport.ErrResourceAlreadyExist, domainModel.Metadata.Name, domainModel.Metadata.Namespace)
	case !errors.Is(err, domainport.ErrResourceNotExist):
		return nil, fmt.Errorf("check existing endpoint: %w", err)
	}

	now := s.clock.Now()

	createdBy, err := security.GetUser(ctx)
	if err != nil {
		s.logger.Warn(
			"failed to get user from context",
			slog.String("error", err.Error()),
		)

		createdBy = security.NewAnonymousUser()
	}

	domainModel.Metadata.CreatedAt = now
	domainModel.Status.Conditions = append(
		domainModel.Status.Conditions,
		model.Condition{
			Type:               model.ConditionTypeCreated,
			Status:             model.ConditionStatusTrue,
			LastTransitionTime: now,
			Reason:             createdBy.String(),
			Message:            "Endpoint created",
		},
	)

	saved, err := s.endpointUsecase.SaveEndpoint(ctx, domainModel)
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
	existing, err := s.endpointUsecase.GetEndpoint(ctx, namespace, name, nil)
	if err != nil {
		return nil, fmt.Errorf("get existing endpoint: %w", err)
	}

	domainModel := s.mapper.MapAPIToEndpoint(apiModel)
	domainModel = s.sanityFilter.Sanitize(existing, domainModel)

	updated, err := s.endpointUsecase.SaveEndpoint(ctx, domainModel)
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
	deletedBy, err := security.GetUser(ctx)
	if err != nil {
		s.logger.Warn(
			"failed to get user from context",
			slog.String("error", err.Error()),
		)

		deletedBy = security.NewAnonymousUser()
	}

	err = s.endpointUsecase.DeleteEndpoint(
		ctx, namespace, name, s.clock.Now(), deletedBy.String(),
	)
	if err != nil {
		return fmt.Errorf("delete endpoint: %w", err)
	}

	return nil
}
