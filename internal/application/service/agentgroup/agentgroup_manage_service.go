// Package agentgroup provides the AgentGroupManageService for managing agent groups.
package agentgroup

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/samber/lo"
	k8sclock "k8s.io/utils/clock"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	v1agent "github.com/minuk-dev/opampcommander/api/v1/agent"
	v1agentgroup "github.com/minuk-dev/opampcommander/api/v1/agentgroup"
	"github.com/minuk-dev/opampcommander/internal/application/mapper"
	"github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
	"github.com/minuk-dev/opampcommander/internal/security"
	"github.com/minuk-dev/opampcommander/pkg/utils/clock"
)

var _ port.AgentGroupManageUsecase = (*ManageService)(nil)

// ManageService implements port.AgentGroupManageUsecase. You can inject repository or other dependencies as needed.
type ManageService struct {
	agentgroupUsecase domainport.AgentGroupUsecase
	agentUsecase      domainport.AgentUsecase
	agentMapper       *mapper.Mapper
	clock             clock.Clock
	logger            *slog.Logger
}

// NewManageService returns a new ManageService.
func NewManageService(
	agentgroupUsecase domainport.AgentGroupUsecase,
	agentUsecase domainport.AgentUsecase,
	logger *slog.Logger,
) *ManageService {
	return &ManageService{
		agentgroupUsecase: agentgroupUsecase,
		agentUsecase:      agentUsecase,
		agentMapper:       mapper.New(),
		clock:             k8sclock.RealClock{},
		logger:            logger,
	}
}

// GetAgentGroup returns an agent group by its UUID.
func (s *ManageService) GetAgentGroup(
	ctx context.Context,
	name string,
) (*v1agentgroup.AgentGroup, error) {
	agentGroup, err := s.agentgroupUsecase.GetAgentGroup(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get agent group: %w", err)
	}

	return s.toAPIModelAgentGroup(agentGroup), nil
}

// ListAgentGroups returns a paginated list of agent groups.
func (s *ManageService) ListAgentGroups(
	ctx context.Context,
	options *model.ListOptions,
) (*v1agentgroup.ListResponse, error) {
	domainResp, err := s.agentgroupUsecase.ListAgentGroups(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("list agent groups: %w", err)
	}

	return v1agentgroup.NewListResponse(
		lo.Map(domainResp.Items, func(agentGroup *model.AgentGroup, _ int) v1agentgroup.AgentGroup {
			return *s.toAPIModelAgentGroup(agentGroup)
		}),
		v1.ListMeta{
			Continue:           domainResp.Continue,
			RemainingItemCount: domainResp.RemainingItemCount,
		},
	), nil
}

// ListAgentsByAgentGroup implements port.AgentGroupManageUsecase.
func (s *ManageService) ListAgentsByAgentGroup(
	ctx context.Context,
	agentGroupName string,
	options *model.ListOptions,
) (*v1agent.ListResponse, error) {
	agentGroup, err := s.agentgroupUsecase.GetAgentGroup(ctx, agentGroupName)
	if err != nil {
		return nil, fmt.Errorf("get agent group: %w", err)
	}

	domainResp, err := s.agentUsecase.ListAgentsBySelector(ctx, agentGroup.Metadata.Selector, options)
	if err != nil {
		return nil, fmt.Errorf("list agents by agent group: %w", err)
	}

	return v1agent.NewListResponse(
		lo.Map(domainResp.Items, func(agent *model.Agent, _ int) v1agent.Agent {
			return *s.agentMapper.MapAgentToAPI(agent)
		}),
		v1.ListMeta{
			Continue:           domainResp.Continue,
			RemainingItemCount: domainResp.RemainingItemCount,
		},
	), nil
}

// CreateAgentGroup creates a new agent group.
func (s *ManageService) CreateAgentGroup(
	ctx context.Context,
	createCommand *port.CreateAgentGroupCommand,
) (*v1agentgroup.AgentGroup, error) {
	requestedBy, err := security.GetUser(ctx)
	if err != nil {
		s.logger.Warn("failed to get user from context", slog.String("error", err.Error()))

		requestedBy = security.NewAnonymousUser()
	}

	domainAgentGroup := s.toDomainModelAgentGroupForCreate(createCommand, requestedBy)

	agentGroup, err := s.agentgroupUsecase.SaveAgentGroup(ctx, createCommand.Name, domainAgentGroup)
	if err != nil {
		return nil, fmt.Errorf("create agent group: %w", err)
	}

	return s.toAPIModelAgentGroup(agentGroup), nil
}

// UpdateAgentGroup updates an existing agent group.
func (s *ManageService) UpdateAgentGroup(
	ctx context.Context,
	name string,
	apiAgentGroup *v1agentgroup.AgentGroup,
) (*v1agentgroup.AgentGroup, error) {
	domainAgentGroup := toDomainModelAgentGroupFromAPI(apiAgentGroup)

	agentGroup, err := s.agentgroupUsecase.SaveAgentGroup(ctx, name, domainAgentGroup)
	if err != nil {
		return nil, fmt.Errorf("update agent group: %w", err)
	}

	return s.toAPIModelAgentGroup(agentGroup), nil
}

// DeleteAgentGroup marks an agent group as deleted.
func (s *ManageService) DeleteAgentGroup(
	ctx context.Context,
	name string,
) error {
	deletedBy, err := security.GetUser(ctx)
	if err != nil {
		s.logger.Warn("failed to get user from context", slog.String("error", err.Error()))

		deletedBy = security.NewAnonymousUser()
	}

	deletedAt := s.clock.Now()

	err = s.agentgroupUsecase.DeleteAgentGroup(ctx, name, deletedAt, deletedBy.String())
	if err != nil {
		return fmt.Errorf("get agent group for delete: %w", err)
	}

	return nil
}

func (s *ManageService) toAPIModelAgentGroup(domain *model.AgentGroup) *v1agentgroup.AgentGroup {
	if domain == nil {
		return nil
	}

	var agentConfig *v1agentgroup.AgentConfig
	if domain.Spec.AgentRemoteConfig != nil {
		//exhaustruct:ignore
		agentConfig = &v1agentgroup.AgentConfig{
			Value:       string(domain.Spec.AgentRemoteConfig.Value),
			ContentType: domain.Spec.AgentRemoteConfig.ContentType,
		}

		// Add connection settings if configured
		if domain.Spec.AgentConnectionConfig != nil {
			agentConfig.ConnectionSettings = s.toAPIConnectionSettings(domain.Spec.AgentConnectionConfig)
		}
	}

	conditions := make([]v1agentgroup.Condition, len(domain.Status.Conditions))
	for i, condition := range domain.Status.Conditions {
		conditions[i] = v1agentgroup.Condition{
			Type:               v1agentgroup.ConditionType(condition.Type),
			LastTransitionTime: condition.LastTransitionTime,
			Status:             v1agentgroup.ConditionStatus(condition.Status),
			Reason:             condition.Reason,
			Message:            condition.Message,
		}
	}

	// Use statistics from domain model (calculated by persistence layer)
	return &v1agentgroup.AgentGroup{
		Metadata: v1agentgroup.Metadata{
			Name:       domain.Metadata.Name,
			Attributes: v1agentgroup.Attributes(domain.Metadata.Attributes),
			Priority:   int(domain.Metadata.Priority),
			Selector: v1agentgroup.AgentSelector{
				IdentifyingAttributes:    domain.Metadata.Selector.IdentifyingAttributes,
				NonIdentifyingAttributes: domain.Metadata.Selector.NonIdentifyingAttributes,
			},
		},
		Spec: v1agentgroup.Spec{
			AgentConfig: agentConfig,
		},
		Status: v1agentgroup.Status{
			NumAgents:             domain.Status.NumAgents,
			NumConnectedAgents:    domain.Status.NumConnectedAgents,
			NumHealthyAgents:      domain.Status.NumHealthyAgents,
			NumUnhealthyAgents:    domain.Status.NumUnhealthyAgents,
			NumNotConnectedAgents: domain.Status.NumNotConnectedAgents,
			Conditions:            conditions,
		},
	}
}

func (s *ManageService) toDomainModelAgentGroupForCreate(
	cmd *port.CreateAgentGroupCommand,
	requestedBy *security.User,
) *model.AgentGroup {
	var agentConfig *model.AgentRemoteConfig
	if cmd.AgentConfig != nil {
		agentConfig = &model.AgentRemoteConfig{
			Value:       []byte(cmd.AgentConfig.Value),
			ContentType: cmd.AgentConfig.ContentType,
		}
	}

	var connectionConfig *model.AgentConnectionConfig
	if cmd.AgentConfig != nil && cmd.AgentConfig.ConnectionSettings != nil {
		connectionConfig = toDomainConnectionConfigFromAPI(cmd.AgentConfig.ConnectionSettings)
	}

	return &model.AgentGroup{
		Metadata: model.AgentGroupMetadata{
			Name:       cmd.Name,
			Attributes: model.Attributes(cmd.Attributes),
			Priority:   int32(cmd.Priority), //nolint:gosec // Priority values are expected to be small
			Selector: model.AgentSelector{
				IdentifyingAttributes:    cmd.Selector.IdentifyingAttributes,
				NonIdentifyingAttributes: cmd.Selector.NonIdentifyingAttributes,
			},
		},
		Spec: model.AgentGroupSpec{
			AgentRemoteConfig:     agentConfig,
			AgentConnectionConfig: connectionConfig,
		},
		Status: model.AgentGroupStatus{
			// No need to set statistics here; they will be calculated by the persistence layer
			NumAgents:             0,
			NumConnectedAgents:    0,
			NumHealthyAgents:      0,
			NumUnhealthyAgents:    0,
			NumNotConnectedAgents: 0,

			Conditions: []model.Condition{
				{
					Type:               model.ConditionTypeCreated,
					LastTransitionTime: s.clock.Now(),
					Status:             model.ConditionStatusTrue,
					Reason:             requestedBy.String(),
					Message:            "Agent group created",
				},
			},
		},
	}
}

func toDomainModelAgentGroupFromAPI(api *v1agentgroup.AgentGroup) *model.AgentGroup {
	if api == nil {
		return nil
	}

	var agentConfig *model.AgentRemoteConfig
	if api.Spec.AgentConfig != nil {
		agentConfig = &model.AgentRemoteConfig{
			Value:       []byte(api.Spec.AgentConfig.Value),
			ContentType: api.Spec.AgentConfig.ContentType,
		}
	}

	var connectionConfig *model.AgentConnectionConfig
	if api.Spec.AgentConfig != nil && api.Spec.AgentConfig.ConnectionSettings != nil {
		connectionConfig = toDomainConnectionConfigFromAPI(api.Spec.AgentConfig.ConnectionSettings)
	}

	conditions := make([]model.Condition, len(api.Status.Conditions))
	for i, condition := range api.Status.Conditions {
		conditions[i] = model.Condition{
			Type:               model.ConditionType(condition.Type),
			LastTransitionTime: condition.LastTransitionTime,
			Status:             model.ConditionStatus(condition.Status),
			Reason:             condition.Reason,
			Message:            condition.Message,
		}
	}

	return &model.AgentGroup{
		Metadata: model.AgentGroupMetadata{
			Name:       api.Metadata.Name,
			Priority:   int32(api.Metadata.Priority), //nolint:gosec // Priority values are expected to be small
			Attributes: model.Attributes(api.Metadata.Attributes),
			Selector: model.AgentSelector{
				IdentifyingAttributes:    api.Metadata.Selector.IdentifyingAttributes,
				NonIdentifyingAttributes: api.Metadata.Selector.NonIdentifyingAttributes,
			},
		},
		Spec: model.AgentGroupSpec{
			AgentRemoteConfig:     agentConfig,
			AgentConnectionConfig: connectionConfig,
		},
		Status: model.AgentGroupStatus{
			NumAgents:             api.Status.NumAgents,
			NumConnectedAgents:    api.Status.NumConnectedAgents,
			NumHealthyAgents:      api.Status.NumHealthyAgents,
			NumUnhealthyAgents:    api.Status.NumUnhealthyAgents,
			NumNotConnectedAgents: api.Status.NumNotConnectedAgents,
			Conditions:            conditions,
		},
	}
}

// toDomainConnectionConfigFromAPI converts API ConnectionSettings to domain AgentConnectionConfig.
func toDomainConnectionConfigFromAPI(
	api *v1agent.ConnectionSettings,
) *model.AgentConnectionConfig {
	if api == nil {
		return nil
	}

	return &model.AgentConnectionConfig{
		OpAMPConnection: model.OpAMPConnectionSettings{
			DestinationEndpoint: api.OpAMP.DestinationEndpoint,
			Headers:             api.OpAMP.Headers,
			Certificate: model.TelemetryTLSCertificate{
				Cert:       []byte(api.OpAMP.Certificate.Cert),
				PrivateKey: []byte(api.OpAMP.Certificate.PrivateKey),
				CaCert:     []byte(api.OpAMP.Certificate.CaCert),
			},
		},
		OwnMetrics: model.TelemetryConnectionSettings{
			DestinationEndpoint: api.OwnMetrics.DestinationEndpoint,
			Headers:             api.OwnMetrics.Headers,
			Certificate: model.TelemetryTLSCertificate{
				Cert:       []byte(api.OwnMetrics.Certificate.Cert),
				PrivateKey: []byte(api.OwnMetrics.Certificate.PrivateKey),
				CaCert:     []byte(api.OwnMetrics.Certificate.CaCert),
			},
		},
		OwnLogs: model.TelemetryConnectionSettings{
			DestinationEndpoint: api.OwnLogs.DestinationEndpoint,
			Headers:             api.OwnLogs.Headers,
			Certificate: model.TelemetryTLSCertificate{
				Cert:       []byte(api.OwnLogs.Certificate.Cert),
				PrivateKey: []byte(api.OwnLogs.Certificate.PrivateKey),
				CaCert:     []byte(api.OwnLogs.Certificate.CaCert),
			},
		},
		OwnTraces: model.TelemetryConnectionSettings{
			DestinationEndpoint: api.OwnTraces.DestinationEndpoint,
			Headers:             api.OwnTraces.Headers,
			Certificate: model.TelemetryTLSCertificate{
				Cert:       []byte(api.OwnTraces.Certificate.Cert),
				PrivateKey: []byte(api.OwnTraces.Certificate.PrivateKey),
				CaCert:     []byte(api.OwnTraces.Certificate.CaCert),
			},
		},
		OtherConnections: toDomainOtherConnectionsFromAPI(api.OtherConnections),
	}
}

// toDomainOtherConnectionsFromAPI converts API other connections to domain format.
func toDomainOtherConnectionsFromAPI(
	api map[string]v1agent.OtherConnectionSettings,
) map[string]model.OtherConnectionSettings {
	if api == nil {
		return nil
	}

	result := make(map[string]model.OtherConnectionSettings, len(api))
	for name, settings := range api {
		result[name] = model.OtherConnectionSettings{
			DestinationEndpoint: settings.DestinationEndpoint,
			Headers:             settings.Headers,
			Certificate: model.TelemetryTLSCertificate{
				Cert:       []byte(settings.Certificate.Cert),
				PrivateKey: []byte(settings.Certificate.PrivateKey),
				CaCert:     []byte(settings.Certificate.CaCert),
			},
		}
	}

	return result
}

// toAPIConnectionSettings converts domain AgentConnectionConfig to API ConnectionSettings.
func (s *ManageService) toAPIConnectionSettings(
	domain *model.AgentConnectionConfig,
) *v1agent.ConnectionSettings {
	if domain == nil {
		return nil
	}

	return &v1agent.ConnectionSettings{
		OpAMP: v1agent.OpAMPConnectionSettings{
			DestinationEndpoint: domain.OpAMPConnection.DestinationEndpoint,
			Headers:             domain.OpAMPConnection.Headers,
			Certificate: v1agent.TLSCertificate{
				Cert:       string(domain.OpAMPConnection.Certificate.Cert),
				PrivateKey: string(domain.OpAMPConnection.Certificate.PrivateKey),
				CaCert:     string(domain.OpAMPConnection.Certificate.CaCert),
			},
		},
		OwnMetrics: v1agent.TelemetryConnectionSettings{
			DestinationEndpoint: domain.OwnMetrics.DestinationEndpoint,
			Headers:             domain.OwnMetrics.Headers,
			Certificate: v1agent.TLSCertificate{
				Cert:       string(domain.OwnMetrics.Certificate.Cert),
				PrivateKey: string(domain.OwnMetrics.Certificate.PrivateKey),
				CaCert:     string(domain.OwnMetrics.Certificate.CaCert),
			},
		},
		OwnLogs: v1agent.TelemetryConnectionSettings{
			DestinationEndpoint: domain.OwnLogs.DestinationEndpoint,
			Headers:             domain.OwnLogs.Headers,
			Certificate: v1agent.TLSCertificate{
				Cert:       string(domain.OwnLogs.Certificate.Cert),
				PrivateKey: string(domain.OwnLogs.Certificate.PrivateKey),
				CaCert:     string(domain.OwnLogs.Certificate.CaCert),
			},
		},
		OwnTraces: v1agent.TelemetryConnectionSettings{
			DestinationEndpoint: domain.OwnTraces.DestinationEndpoint,
			Headers:             domain.OwnTraces.Headers,
			Certificate: v1agent.TLSCertificate{
				Cert:       string(domain.OwnTraces.Certificate.Cert),
				PrivateKey: string(domain.OwnTraces.Certificate.PrivateKey),
				CaCert:     string(domain.OwnTraces.Certificate.CaCert),
			},
		},
		OtherConnections: s.toAPIOtherConnections(domain.OtherConnections),
	}
}

// toAPIOtherConnections converts domain other connections to API format.
func (s *ManageService) toAPIOtherConnections(
	domain map[string]model.OtherConnectionSettings,
) map[string]v1agent.OtherConnectionSettings {
	if domain == nil {
		return nil
	}

	result := make(map[string]v1agent.OtherConnectionSettings, len(domain))
	for name, settings := range domain {
		result[name] = v1agent.OtherConnectionSettings{
			DestinationEndpoint: settings.DestinationEndpoint,
			Headers:             settings.Headers,
			Certificate: v1agent.TLSCertificate{
				Cert:       string(settings.Certificate.Cert),
				PrivateKey: string(settings.Certificate.PrivateKey),
				CaCert:     string(settings.Certificate.CaCert),
			},
		}
	}

	return result
}
