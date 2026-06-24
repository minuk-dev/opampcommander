// Package agentremoteconfig provides the service for managing agent remote configs.
package agentremoteconfig

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/helper"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/service/agentremoteconfig/filter"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/security"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var _ port.AgentRemoteConfigManageUsecase = (*Service)(nil)

// Service is a service for managing agent remote configs.
type Service struct {
	agentRemoteConfigUsecase agentport.AgentRemoteConfigUsecase
	agentGroupUsecase        agentport.AgentGroupUsecase
	endpointDetectionUsecase agentport.EndpointDetectionUsecase
	mapper                   *helper.Mapper
	sanityFilter             *filter.Sanity
	clock                    clock.Clock
	logger                   *slog.Logger
}

// NewAgentRemoteConfigService creates a new AgentRemoteConfigService.
func NewAgentRemoteConfigService(
	agentRemoteConfigUsecase agentport.AgentRemoteConfigUsecase,
	agentGroupUsecase agentport.AgentGroupUsecase,
	endpointDetectionUsecase agentport.EndpointDetectionUsecase,
	logger *slog.Logger,
) *Service {
	realClock := clock.NewRealClock()

	return &Service{
		agentRemoteConfigUsecase: agentRemoteConfigUsecase,
		agentGroupUsecase:        agentGroupUsecase,
		endpointDetectionUsecase: endpointDetectionUsecase,
		mapper:                   helper.NewMapper(realClock, 0),
		sanityFilter:             filter.NewSanity(),
		clock:                    realClock,
		logger:                   logger,
	}
}

// GetAgentRemoteConfig implements [port.AgentRemoteConfigManageUsecase].
func (s *Service) GetAgentRemoteConfig(
	ctx context.Context,
	namespace string,
	name string,
	options *port.GetOptions,
) (*v1.AgentRemoteConfig, error) {
	config, err := s.agentRemoteConfigUsecase.GetAgentRemoteConfig(
		ctx, namespace, name, options.ToDomain(),
	)
	if err != nil {
		return nil, fmt.Errorf("get agent remote config: %w", err)
	}

	return s.mapper.MapAgentRemoteConfigToAPI(config), nil
}

// ListAgentRemoteConfigs implements [port.AgentRemoteConfigManageUsecase].
func (s *Service) ListAgentRemoteConfigs(
	ctx context.Context,
	options *port.ListOptions,
) (*v1.ListResponse[v1.AgentRemoteConfig], error) {
	configs, err := s.agentRemoteConfigUsecase.ListAgentRemoteConfigs(
		ctx, options.ToDomain(),
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

	s.triggerGroupPropagation(ctx, saved.Metadata.Namespace, saved.Metadata.Name)
	s.triggerEndpointDetection(ctx, saved)

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
		ctx, namespace, name, nil,
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

	s.triggerGroupPropagation(ctx, updated.Metadata.Namespace, updated.Metadata.Name)
	s.triggerEndpointDetection(ctx, updated)

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

	s.triggerGroupPropagation(ctx, namespace, name)

	return nil
}

// ReconcileAgentRemoteConfig implements [port.AgentRemoteConfigManageUsecase]. It delegates
// to the domain use case, which re-detects endpoints from the config's collector exporters
// and re-propagates the config to the agent groups that reference it. Unlike the create/update
// path (which fires these as detached goroutines), this runs synchronously and surfaces errors.
func (s *Service) ReconcileAgentRemoteConfig(ctx context.Context, namespace string, name string) error {
	err := s.agentRemoteConfigUsecase.ReconcileAgentRemoteConfig(ctx, namespace, name)
	if err != nil {
		return fmt.Errorf("reconcile agent remote config: %w", err)
	}

	return nil
}

// triggerGroupPropagation asks the agent group service to re-apply any groups in the
// namespace that reference this config. Runs in its own goroutine so a slow
// changedAgentGroupCh consumer cannot block the HTTP handler; the periodic
// reconciliation loop is the durable safety net.
//
// We detach the goroutine from the request ctx with context.WithoutCancel so the
// propagation finishes even if the HTTP request is cancelled right after mutation.
func (s *Service) triggerGroupPropagation(ctx context.Context, namespace, name string) {
	bgCtx := context.WithoutCancel(ctx)

	go func() {
		err := s.agentGroupUsecase.PropagateAgentRemoteConfigChange(bgCtx, namespace, name)
		if err != nil {
			s.logger.Warn(
				"failed to propagate agent remote config change to groups",
				slog.String("namespace", namespace),
				slog.String("name", name),
				slog.String("error", err.Error()),
			)
		}
	}()
}

// triggerEndpointDetection reconciles the endpoints derived from this remote
// config's collector configuration. Runs detached from the request ctx, in its own
// goroutine, so it cannot block (or be cancelled with) the HTTP handler.
func (s *Service) triggerEndpointDetection(ctx context.Context, remoteConfig *agentmodel.AgentRemoteConfig) {
	bgCtx := context.WithoutCancel(ctx)

	go func() {
		err := s.endpointDetectionUsecase.ReconcileEndpointsFromRemoteConfig(bgCtx, remoteConfig)
		if err != nil {
			s.logger.Warn(
				"failed to reconcile endpoints from remote config",
				slog.String("namespace", remoteConfig.Metadata.Namespace),
				slog.String("name", remoteConfig.Metadata.Name),
				slog.String("error", err.Error()),
			)
		}
	}()
}
