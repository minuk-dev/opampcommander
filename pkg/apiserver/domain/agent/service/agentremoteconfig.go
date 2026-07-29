package agentservice

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var _ agentport.AgentRemoteConfigUsecase = (*AgentRemoteConfigService)(nil)

// AgentRemoteConfigService provides operations for managing agent remote configs,
// including the creation/update lifecycle rules (stamping and immutable-field
// preservation).
type AgentRemoteConfigService struct {
	persistence agentport.AgentRemoteConfigPersistencePort

	// other domain usecases, used to re-run a config's side effects on reconcile.
	endpointDetectionUsecase agentport.EndpointDetectionUsecase
	agentGroupUsecase        agentport.AgentGroupUsecase
	// schemaMatcher auto-detects which RemoteConfigSchemas a config is compatible
	// with, to populate SchemaRefs when the caller did not set them. May be nil.
	schemaMatcher agentport.RemoteConfigSchemaMatcher

	clock  clock.Clock
	logger *slog.Logger
}

// NewAgentRemoteConfigService creates a new AgentRemoteConfigService.
func NewAgentRemoteConfigService(
	persistence agentport.AgentRemoteConfigPersistencePort,
	endpointDetectionUsecase agentport.EndpointDetectionUsecase,
	agentGroupUsecase agentport.AgentGroupUsecase,
	schemaMatcher agentport.RemoteConfigSchemaMatcher,
	logger *slog.Logger,
) *AgentRemoteConfigService {
	return &AgentRemoteConfigService{
		persistence:              persistence,
		endpointDetectionUsecase: endpointDetectionUsecase,
		agentGroupUsecase:        agentGroupUsecase,
		schemaMatcher:            schemaMatcher,
		clock:                    clock.NewRealClock(),
		logger:                   logger,
	}
}

// SetClock overrides the clock used for lifecycle timestamps. Intended for tests.
func (s *AgentRemoteConfigService) SetClock(c clock.Clock) {
	s.clock = c
}

// GetAgentRemoteConfig implements [agentport.AgentRemoteConfigUsecase].
func (s *AgentRemoteConfigService) GetAgentRemoteConfig(
	ctx context.Context,
	namespace string,
	name string,
	options *model.GetOptions,
) (*agentmodel.AgentRemoteConfig, error) {
	resource, err := s.persistence.GetAgentRemoteConfig(ctx, namespace, name, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent remote config: %w", err)
	}

	// Convert resource to the simpler AgentRemoteConfig type
	return resource, nil
}

// ListAgentRemoteConfigs implements [agentport.AgentRemoteConfigUsecase].
func (s *AgentRemoteConfigService) ListAgentRemoteConfigs(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentRemoteConfig], error) {
	resourceResp, err := s.persistence.ListAgentRemoteConfigs(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list agent remote configs: %w", err)
	}

	return &model.ListResponse[*agentmodel.AgentRemoteConfig]{
		Items:              resourceResp.Items,
		RemainingItemCount: resourceResp.RemainingItemCount,
		Continue:           resourceResp.Continue,
	}, nil
}

// SaveAgentRemoteConfig implements [agentport.AgentRemoteConfigUsecase].
func (s *AgentRemoteConfigService) SaveAgentRemoteConfig(
	ctx context.Context,
	agentremoteconfig *agentmodel.AgentRemoteConfig,
) (*agentmodel.AgentRemoteConfig, error) {
	resource, err := s.persistence.PutAgentRemoteConfig(ctx, agentremoteconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to save agent remote config: %w", err)
	}

	// Convert resource to the simpler AgentRemoteConfig type
	return resource, nil
}

// CreateAgentRemoteConfig implements [agentport.AgentRemoteConfigUsecase].
func (s *AgentRemoteConfigService) CreateAgentRemoteConfig(
	ctx context.Context,
	agentRemoteConfig *agentmodel.AgentRemoteConfig,
	actor string,
) (*agentmodel.AgentRemoteConfig, error) {
	agentRemoteConfig.MarkAsCreated(s.clock.Now(), actor)
	s.autoResolveSchemaRefs(ctx, agentRemoteConfig)

	created, err := s.persistence.PutAgentRemoteConfig(ctx, agentRemoteConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent remote config: %w", err)
	}

	return created, nil
}

// UpdateAgentRemoteConfig implements [agentport.AgentRemoteConfigUsecase].
func (s *AgentRemoteConfigService) UpdateAgentRemoteConfig(
	ctx context.Context,
	namespace string,
	name string,
	agentRemoteConfig *agentmodel.AgentRemoteConfig,
) (*agentmodel.AgentRemoteConfig, error) {
	existing, err := s.persistence.GetAgentRemoteConfig(ctx, namespace, name, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent remote config for update: %w", err)
	}

	existing.ApplyUpdate(agentRemoteConfig)

	updated, err := s.persistence.PutAgentRemoteConfig(ctx, existing)
	if err != nil {
		return nil, fmt.Errorf("failed to update agent remote config: %w", err)
	}

	return updated, nil
}

// DeleteAgentRemoteConfig implements [agentport.AgentRemoteConfigUsecase].
func (s *AgentRemoteConfigService) DeleteAgentRemoteConfig(
	ctx context.Context,
	namespace string,
	name string,
	deletedAt time.Time,
	deletedBy string,
) error {
	resource, err := s.persistence.GetAgentRemoteConfig(ctx, namespace, name, nil)
	if err != nil {
		return fmt.Errorf("failed to get agent remote config for deletion: %w", err)
	}

	resource.MarkDeleted(deletedAt, deletedBy)

	_, err = s.persistence.PutAgentRemoteConfig(ctx, resource)
	if err != nil {
		return fmt.Errorf("failed to mark agent remote config as deleted: %w", err)
	}

	return nil
}

// ReconcileAgentRemoteConfig implements [agentport.AgentRemoteConfigUsecase]. It loads the
// named config and re-runs the side effects that normally fire on create/update: telemetry
// endpoint detection from the config's collector exporters, then re-propagation of the config
// to every agent group that references it. Both run synchronously so the caller learns of
// failures (unlike the fire-and-forget triggers on the write path).
func (s *AgentRemoteConfigService) ReconcileAgentRemoteConfig(
	ctx context.Context,
	namespace string,
	name string,
) error {
	remoteConfig, err := s.persistence.GetAgentRemoteConfig(ctx, namespace, name, nil)
	if err != nil {
		return fmt.Errorf("get agent remote config %s/%s: %w", namespace, name, err)
	}

	err = s.endpointDetectionUsecase.ReconcileEndpointsFromRemoteConfig(ctx, remoteConfig)
	if err != nil {
		return fmt.Errorf("reconcile endpoints from remote config %s/%s: %w", namespace, name, err)
	}

	err = s.agentGroupUsecase.PropagateAgentRemoteConfigChange(ctx, namespace, name)
	if err != nil {
		return fmt.Errorf("propagate remote config change %s/%s: %w", namespace, name, err)
	}

	return nil
}

// autoResolveSchemaRefs fills config.Spec.SchemaRefs with the schemas the config is
// compatible with, when the caller left it empty and a matcher is available. It runs
// only on create, so an update can explicitly clear SchemaRefs without them being
// re-derived, and is skipped entirely when the config carries the
// SkipSchemaValidationAnnotation. It is best-effort: a resolution error is logged but
// never blocks the save (an explicit SchemaRefs is always preserved).
func (s *AgentRemoteConfigService) autoResolveSchemaRefs(
	ctx context.Context,
	config *agentmodel.AgentRemoteConfig,
) {
	if s.schemaMatcher == nil || len(config.Spec.SchemaRefs) > 0 || config.SkipSchemaValidation() {
		return
	}

	refs, err := s.schemaMatcher.ResolveSchemaRefs(ctx, config)
	if err != nil {
		if s.logger != nil {
			s.logger.WarnContext(ctx, "failed to auto-resolve schema refs",
				slog.String("namespace", config.Metadata.Namespace),
				slog.String("name", config.Metadata.Name),
				slog.String("error", err.Error()))
		}

		return
	}

	if len(refs) > 0 {
		config.Spec.SchemaRefs = refs
	}
}
