// Package agentremoteconfig provides the service for managing agent remote configs.
package agentremoteconfig

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/internal/application/helper"
	"github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/application/service/agentremoteconfig/filter"
	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/internal/domain/agent/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/security"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var _ port.AgentRemoteConfigManageUsecase = (*Service)(nil)

// Service is a service for managing agent remote configs.
type Service struct {
	agentRemoteConfigUsecase agentport.AgentRemoteConfigUsecase
	mapper                   *helper.Mapper
	sanityFilter             *filter.Sanity
	clock                    clock.Clock
	logger                   *slog.Logger
}

// NewAgentRemoteConfigService creates a new AgentRemoteConfigService.
func NewAgentRemoteConfigService(
	agentRemoteConfigUsecase agentport.AgentRemoteConfigUsecase,
	logger *slog.Logger,
) *Service {
	return &Service{
		agentRemoteConfigUsecase: agentRemoteConfigUsecase,
		mapper:                   helper.NewMapper(),
		sanityFilter:             filter.NewSanity(),
		clock:                    clock.NewRealClock(),
		logger:                   logger,
	}
}

// GetAgentRemoteConfig implements [port.AgentRemoteConfigManageUsecase].
func (s *Service) GetAgentRemoteConfig(
	ctx context.Context,
	namespace string,
	name string,
) (*v1.AgentRemoteConfig, error) {
	config, err := s.agentRemoteConfigUsecase.GetAgentRemoteConfig(
		ctx, namespace, name,
	)
	if err != nil {
		return nil, fmt.Errorf("get agent remote config: %w", err)
	}

	return s.mapper.MapAgentRemoteConfigToAPI(config), nil
}

// ListAgentRemoteConfigs implements [port.AgentRemoteConfigManageUsecase].
func (s *Service) ListAgentRemoteConfigs(
	ctx context.Context,
	options *model.ListOptions,
) (*v1.ListResponse[v1.AgentRemoteConfig], error) {
	configs, err := s.agentRemoteConfigUsecase.ListAgentRemoteConfigs(
		ctx, options,
	)
	if err != nil {
		return nil, fmt.Errorf("list agent remote configs: %w", err)
	}

	return &v1.ListResponse[v1.AgentRemoteConfig]{
		Kind:       v1.AgentRemoteConfigKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.ListMeta{
			Continue:           configs.Continue,
			RemainingItemCount: configs.RemainingItemCount,
		},
		Items: lo.Map(
			configs.Items,
			func(
				item *agentmodel.AgentRemoteConfig, _ int,
			) v1.AgentRemoteConfig {
				return *s.mapper.MapAgentRemoteConfigToAPI(item)
			},
		),
	}, nil
}

// CreateAgentRemoteConfig implements [port.AgentRemoteConfigManageUsecase].
func (s *Service) CreateAgentRemoteConfig(
	ctx context.Context,
	apiModel *v1.AgentRemoteConfig,
) (*v1.AgentRemoteConfig, error) {
	domainModel := s.mapper.MapAPIToAgentRemoteConfig(apiModel)

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
			Message:            "Agent remote config created",
		},
	)

	saved, err := s.agentRemoteConfigUsecase.SaveAgentRemoteConfig(
		ctx, domainModel,
	)
	if err != nil {
		return nil, fmt.Errorf("create agent remote config: %w", err)
	}

	return s.mapper.MapAgentRemoteConfigToAPI(saved), nil
}

// UpdateAgentRemoteConfig implements [port.AgentRemoteConfigManageUsecase].
func (s *Service) UpdateAgentRemoteConfig(
	ctx context.Context,
	namespace string,
	name string,
	apiModel *v1.AgentRemoteConfig,
) (*v1.AgentRemoteConfig, error) {
	existing, err := s.agentRemoteConfigUsecase.GetAgentRemoteConfig(
		ctx, namespace, name,
	)
	if err != nil {
		return nil, fmt.Errorf("get existing agent remote config: %w", err)
	}

	domainModel := s.mapper.MapAPIToAgentRemoteConfig(apiModel)
	domainModel = s.sanityFilter.Sanitize(existing, domainModel)

	updated, err := s.agentRemoteConfigUsecase.SaveAgentRemoteConfig(
		ctx, domainModel,
	)
	if err != nil {
		return nil, fmt.Errorf("update agent remote config: %w", err)
	}

	return s.mapper.MapAgentRemoteConfigToAPI(updated), nil
}

// DeleteAgentRemoteConfig implements [port.AgentRemoteConfigManageUsecase].
func (s *Service) DeleteAgentRemoteConfig(
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

	err = s.agentRemoteConfigUsecase.DeleteAgentRemoteConfig(
		ctx, namespace, name, s.clock.Now(), deletedBy.String(),
	)
	if err != nil {
		return fmt.Errorf("delete agent remote config: %w", err)
	}

	return nil
}
