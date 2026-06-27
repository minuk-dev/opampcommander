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
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/security"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var _ port.AgentRemoteConfigManageUsecase = (*Service)(nil)

// Service is a service for managing agent remote configs. It maps between the HTTP
// DTOs and the domain, resolves the acting user, delegates lifecycle rules
// (stamping, immutable-field preservation) to the domain AgentRemoteConfigUsecase,
// and fans the post-write side effects (group propagation, endpoint detection) out
// to the relevant domain usecases without blocking the request.
type Service struct {
	agentRemoteConfigUsecase agentport.AgentRemoteConfigUsecase
	agentGroupUsecase        agentport.AgentGroupUsecase
	endpointDetectionUsecase agentport.EndpointDetectionUsecase
	mapper                   *helper.Mapper
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

	saved, err := s.agentRemoteConfigUsecase.CreateAgentRemoteConfig(ctx, domainModel, s.actor(ctx))
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
	domainModel := s.mapper.MapAPIToAgentRemoteConfig(apiModel)

	updated, err := s.agentRemoteConfigUsecase.UpdateAgentRemoteConfig(
		ctx, namespace, name, domainModel,
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
	err := s.agentRemoteConfigUsecase.DeleteAgentRemoteConfig(
		ctx, namespace, name, s.clock.Now(), s.actor(ctx),
	)
	if err != nil {
		return fmt.Errorf("delete agent remote config: %w", err)
	}

	s.triggerGroupPropagation(ctx, namespace, name)

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
