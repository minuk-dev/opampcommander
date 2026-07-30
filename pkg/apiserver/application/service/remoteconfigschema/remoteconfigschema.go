// Package remoteconfigschema provides the service for managing remote config schemas.
package remoteconfigschema

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/helper"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/usecase"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/security"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var _ usecase.RemoteConfigSchemaManageUsecase = (*Service)(nil)

// Service maps between the HTTP DTOs and the domain, resolves the acting user, and
// delegates all lifecycle rules to the domain RemoteConfigSchemaUsecase.
type Service struct {
	schemaUsecase agentport.RemoteConfigSchemaUsecase
	mapper        *helper.Mapper
	clock         clock.Clock
	logger        *slog.Logger
}

// NewRemoteConfigSchemaService creates a new remote config schema Service.
func NewRemoteConfigSchemaService(
	schemaUsecase agentport.RemoteConfigSchemaUsecase,
	logger *slog.Logger,
) *Service {
	realClock := clock.NewRealClock()

	return &Service{
		schemaUsecase: schemaUsecase,
		mapper:        helper.NewMapper(realClock, 0),
		clock:         realClock,
		logger:        logger,
	}
}

// GetRemoteConfigSchema implements [usecase.RemoteConfigSchemaManageUsecase].
func (s *Service) GetRemoteConfigSchema(
	ctx context.Context,
	namespace string,
	name string,
	options *port.GetOptions,
) (*v1.RemoteConfigSchema, error) {
	schema, err := s.schemaUsecase.GetRemoteConfigSchema(ctx, namespace, name, options.ToDomain())
	if err != nil {
		return nil, fmt.Errorf("get remote config schema: %w", err)
	}

	return s.mapper.MapRemoteConfigSchemaToAPI(schema), nil
}

// ListRemoteConfigSchemas implements [usecase.RemoteConfigSchemaManageUsecase].
func (s *Service) ListRemoteConfigSchemas(
	ctx context.Context,
	namespace string,
	options *port.ListOptions,
) (*v1.ListResponse[v1.RemoteConfigSchema], error) {
	schemas, err := s.schemaUsecase.ListRemoteConfigSchemas(ctx, namespace, options.ToDomain())
	if err != nil {
		return nil, fmt.Errorf("list remote config schemas: %w", err)
	}

	return &v1.ListResponse[v1.RemoteConfigSchema]{
		Kind:       v1.RemoteConfigSchemaKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.ListMeta{
			Continue:           schemas.Continue,
			RemainingItemCount: schemas.RemainingItemCount,
		},
		Items: lo.Map(
			schemas.Items,
			func(item *agentmodel.RemoteConfigSchema, _ int) v1.RemoteConfigSchema {
				return *s.mapper.MapRemoteConfigSchemaToAPI(item)
			},
		),
	}, nil
}

// CreateRemoteConfigSchema implements [usecase.RemoteConfigSchemaManageUsecase].
func (s *Service) CreateRemoteConfigSchema(
	ctx context.Context,
	apiModel *v1.RemoteConfigSchema,
) (*v1.RemoteConfigSchema, error) {
	domainModel := s.mapper.MapAPIToRemoteConfigSchema(apiModel)

	saved, err := s.schemaUsecase.CreateRemoteConfigSchema(ctx, domainModel, s.actor(ctx))
	if err != nil {
		return nil, fmt.Errorf("create remote config schema: %w", err)
	}

	return s.mapper.MapRemoteConfigSchemaToAPI(saved), nil
}

// UpdateRemoteConfigSchema implements [usecase.RemoteConfigSchemaManageUsecase].
func (s *Service) UpdateRemoteConfigSchema(
	ctx context.Context,
	namespace string,
	name string,
	apiModel *v1.RemoteConfigSchema,
) (*v1.RemoteConfigSchema, error) {
	domainModel := s.mapper.MapAPIToRemoteConfigSchema(apiModel)

	updated, err := s.schemaUsecase.UpdateRemoteConfigSchema(ctx, namespace, name, domainModel)
	if err != nil {
		return nil, fmt.Errorf("update remote config schema: %w", err)
	}

	return s.mapper.MapRemoteConfigSchemaToAPI(updated), nil
}

// DeleteRemoteConfigSchema implements [usecase.RemoteConfigSchemaManageUsecase].
func (s *Service) DeleteRemoteConfigSchema(
	ctx context.Context,
	namespace string,
	name string,
) error {
	err := s.schemaUsecase.DeleteRemoteConfigSchema(
		ctx, namespace, name, s.clock.Now(), s.actor(ctx),
	)
	if err != nil {
		return fmt.Errorf("delete remote config schema: %w", err)
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
