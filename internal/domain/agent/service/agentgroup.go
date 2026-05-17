package agentservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/agent/model"
	agentport "github.com/minuk-dev/opampcommander/internal/domain/agent/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

const (
	agentGroupServiceName = "AgentGroupService"
	// ChangedAgentGroupBufferSize is the buffer size for the changed agent group channel.
	ChangedAgentGroupBufferSize = 100
	// PropagationChunkSize is the number of agents to process in each batch when propagating changes.
	PropagationChunkSize = 50
)

// ErrInvalidRemoteConfig is returned when inline remote config is missing required fields.
var ErrInvalidRemoteConfig = errors.New("invalid remote config: both spec and name are required for inline config")

var _ agentport.AgentGroupUsecase = (*AgentGroupService)(nil)
var _ agentport.AgentGroupRelatedUsecase = (*AgentGroupService)(nil)

// AgentGroupService is a struct that implements the AgentGroupUsecase interface.
type AgentGroupService struct {
	// main port
	persistencePort agentport.AgentGroupPersistencePort

	// related port
	remoteConfigPersistencePort agentport.AgentRemoteConfigPersistencePort
	certificatePersistencePort  agentport.CertificatePersistencePort

	// other domain usecases
	agentUsecase agentport.AgentUsecase

	// internalStatus
	changedAgentGroupCh chan *agentmodel.AgentGroup

	// utils
	logger *slog.Logger
}

// NewAgentGroupService creates a new instance of AgentGroupService.
func NewAgentGroupService(
	persistencePort agentport.AgentGroupPersistencePort,
	agentRemoteConfigPersistencePort agentport.AgentRemoteConfigPersistencePort,
	certificatePersistencePort agentport.CertificatePersistencePort,
	agentUsecase agentport.AgentUsecase,
	logger *slog.Logger,
) *AgentGroupService {
	return &AgentGroupService{
		persistencePort:             persistencePort,
		remoteConfigPersistencePort: agentRemoteConfigPersistencePort,
		certificatePersistencePort:  certificatePersistencePort,
		agentUsecase:                agentUsecase,
		logger:                      logger,
		changedAgentGroupCh:         make(chan *agentmodel.AgentGroup, ChangedAgentGroupBufferSize),
	}
}

// Name implements lifecycle.Runner.
func (s *AgentGroupService) Name() string {
	return agentGroupServiceName
}

// Run implements lifecycle.Runner.
func (s *AgentGroupService) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case agentGroup := <-s.changedAgentGroupCh:
			err := s.updateAgentsByAgentGroup(ctx, agentGroup)
			if err != nil {
				s.logger.Error("failed to propagate agent group changes to agents",
					slog.String("agent_group", agentGroup.Metadata.Name),
					slog.String("error", err.Error()),
				)
			}
		}
	}
	// unreachable
}

// GetAgentGroup retrieves an agent group by its namespace and name.
func (s *AgentGroupService) GetAgentGroup(
	ctx context.Context,
	namespace string,
	name string,
	options *model.GetOptions,
) (*agentmodel.AgentGroup, error) {
	agentGroup, err := s.persistencePort.GetAgentGroup(ctx, namespace, name, options)
	if err != nil {
		return nil, fmt.Errorf("get agent group: %w", err)
	}

	return agentGroup, nil
}

// SaveAgentGroup saves the agent group.
func (s *AgentGroupService) SaveAgentGroup(
	ctx context.Context,
	namespace string,
	name string,
	agentGroup *agentmodel.AgentGroup,
) (*agentmodel.AgentGroup, error) {
	agentGroup, err := s.persistencePort.PutAgentGroup(ctx, namespace, name, agentGroup)
	if err != nil {
		return nil, fmt.Errorf("save agent group: %w", err)
	}

	err = s.propagateAgentGroupChangesToAgents(ctx, agentGroup)
	if err != nil {
		return nil, fmt.Errorf("propagate agent group changes to agents: %w", err)
	}

	return agentGroup, nil
}

// ListAgentGroups retrieves a list of agent groups with pagination options.
func (s *AgentGroupService) ListAgentGroups(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.AgentGroup], error) {
	resp, err := s.persistencePort.ListAgentGroups(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("list agent groups: %w", err)
	}

	return resp, nil
}

// DeleteAgentGroup marks an agent group as deleted.
func (s *AgentGroupService) DeleteAgentGroup(
	ctx context.Context,
	namespace string,
	name string,
	deletedAt time.Time,
	deletedBy string,
) error {
	agentGroup, err := s.persistencePort.GetAgentGroup(ctx, namespace, name, nil)
	if err != nil {
		return fmt.Errorf("failed to get agent group: %w", err)
	}

	agentGroup.MarkDeleted(deletedAt, deletedBy)

	_, err = s.persistencePort.PutAgentGroup(ctx, namespace, name, agentGroup)
	if err != nil {
		return fmt.Errorf("failed to delete agent group: %w", err)
	}

	return nil
}

// ListAgentsByAgentGroup lists agents that belong to the specified agent group.
func (s *AgentGroupService) ListAgentsByAgentGroup(
	ctx context.Context,
	agentGroup *agentmodel.AgentGroup,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Agent], error) {
	agentSelector := agentGroup.Spec.Selector

	listResp, err := s.agentUsecase.ListAgentsBySelector(
		ctx,
		agentSelector,
		options,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents by agent group: %w", err)
	}

	return listResp, nil
}

// GetAgentGroupsForAgent retrieves all agent groups that match the agent's attributes.
func (s *AgentGroupService) GetAgentGroupsForAgent(
	ctx context.Context,
	agent *agentmodel.Agent,
) ([]*agentmodel.AgentGroup, error) {
	// Get all agent groups
	allGroups, err := s.persistencePort.ListAgentGroups(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list agent groups: %w", err)
	}

	// Filter groups that match the agent
	var matchingGroups []*agentmodel.AgentGroup

	for _, group := range allGroups.Items {
		if group.IsDeleted() {
			continue
		}

		if matchesSelector(agent, group.Spec.Selector) {
			matchingGroups = append(matchingGroups, group)
		}
	}

	return matchingGroups, nil
}

// matchesSelector checks if an agent matches the given selector.
func matchesSelector(agent *agentmodel.Agent, selector agentmodel.AgentSelector) bool {
	// Check identifying attributes
	for key, value := range selector.IdentifyingAttributes {
		agentValue, ok := agent.Metadata.Description.IdentifyingAttributes[key]
		if !ok || agentValue != value {
			return false
		}
	}

	// Check non-identifying attributes
	for key, value := range selector.NonIdentifyingAttributes {
		agentValue, ok := agent.Metadata.Description.NonIdentifyingAttributes[key]
		if !ok || agentValue != value {
			return false
		}
	}

	return true
}

func (s *AgentGroupService) propagateAgentGroupChangesToAgents(
	ctx context.Context,
	agentGroup *agentmodel.AgentGroup,
) error {
	select {
	case s.changedAgentGroupCh <- agentGroup:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context cancelled: %w", ctx.Err())
	}
}

func (s *AgentGroupService) updateAgentsByAgentGroup(
	ctx context.Context,
	agentGroup *agentmodel.AgentGroup,
) error {
	var continueToken string

	for {
		agentsResp, err := s.ListAgentsByAgentGroup(ctx, agentGroup, &model.ListOptions{
			Limit:          PropagationChunkSize,
			Continue:       continueToken,
			IncludeDeleted: false,
		})
		if err != nil {
			return fmt.Errorf("list agents by agent group: %w", err)
		}

		if len(agentsResp.Items) == 0 {
			break
		}

		for _, agent := range agentsResp.Items {
			err := s.applyAgentGroupToAgent(ctx, agentGroup, agent)
			if err != nil {
				return fmt.Errorf("apply agent group to agent %s: %w", agent.Metadata.InstanceUID, err)
			}

			err = s.agentUsecase.SaveAgent(ctx, agent)
			if err != nil {
				return fmt.Errorf("save updated agent: %w", err)
			}
		}

		// No more pages to fetch
		if agentsResp.Continue == "" {
			break
		}

		continueToken = agentsResp.Continue
	}

	return nil
}

func (s *AgentGroupService) applyAgentGroupToAgent(
	ctx context.Context,
	agentGroup *agentmodel.AgentGroup,
	agent *agentmodel.Agent,
) error {
	err := s.applyRemoteConfigs(ctx, agentGroup, agent)
	if err != nil {
		return err
	}

	err = s.applyConnectionSettings(ctx, agentGroup, agent)
	if err != nil {
		return err
	}

	return nil
}

func (s *AgentGroupService) applyRemoteConfigs(
	ctx context.Context,
	agentGroup *agentmodel.AgentGroup,
	agent *agentmodel.Agent,
) error {
	agentGroupName := agentGroup.Metadata.Name
	namespace := agentGroup.Metadata.Namespace

	// Handle single remote config (API compatibility)
	//nolint:staticcheck // backward compatibility - AgentRemoteConfig is deprecated
	if agentGroup.Spec.AgentRemoteConfig != nil {
		//nolint:staticcheck // backward compatibility
		configFile, configName, err := s.resolveRemoteConfig(
			ctx, namespace, agentGroupName, *agentGroup.Spec.AgentRemoteConfig)
		if err != nil {
			return err
		}

		err = agent.ApplyRemoteConfig(configName, configFile)
		if err != nil {
			return fmt.Errorf("apply remote config %s: %w", configName, err)
		}
	}

	// Handle multiple remote configs
	for _, remoteConfig := range agentGroup.Spec.AgentRemoteConfigs {
		configFile, configName, err := s.resolveRemoteConfig(ctx, namespace, agentGroupName, remoteConfig)
		if err != nil {
			return err
		}

		err = agent.ApplyRemoteConfig(configName, configFile)
		if err != nil {
			return fmt.Errorf("apply remote config %s: %w", configName, err)
		}
	}

	return nil
}

func (s *AgentGroupService) resolveRemoteConfig(
	ctx context.Context,
	namespace string,
	agentGroupName string,
	remoteConfig agentmodel.AgentGroupAgentRemoteConfig,
) (agentmodel.AgentConfigFile, string, error) {
	// Case 1: Reference to existing AgentRemoteConfig resource
	if remoteConfig.AgentRemoteConfigRef != nil {
		arc, err := s.remoteConfigPersistencePort.GetAgentRemoteConfig(
			ctx, namespace, *remoteConfig.AgentRemoteConfigRef, nil)
		if err != nil {
			return agentmodel.AgentConfigFile{}, "", fmt.Errorf("get agent remote config %s: %w",
				*remoteConfig.AgentRemoteConfigRef, err)
		}

		// Use the original resource name (no prefix needed for refs)
		return agentmodel.AgentConfigFile{
			Body:        arc.Spec.Value,
			ContentType: arc.Spec.ContentType,
		}, arc.Metadata.Name, nil
	}

	// Case 2: Inline/direct config definition
	if remoteConfig.AgentRemoteConfigSpec == nil || remoteConfig.AgentRemoteConfigName == nil {
		return agentmodel.AgentConfigFile{}, "", ErrInvalidRemoteConfig
	}

	// Prefix with AgentGroupName to avoid name collisions
	// Format: {AgentGroupName}/{AgentRemoteConfigName}
	prefixedName := fmt.Sprintf("%s/%s", agentGroupName, *remoteConfig.AgentRemoteConfigName)

	return agentmodel.AgentConfigFile{
		Body:        remoteConfig.AgentRemoteConfigSpec.Value,
		ContentType: remoteConfig.AgentRemoteConfigSpec.ContentType,
	}, prefixedName, nil
}

func (s *AgentGroupService) applyConnectionSettings(
	ctx context.Context,
	agentGroup *agentmodel.AgentGroup,
	agent *agentmodel.Agent,
) error {
	logger := s.logger.With(
		slog.String("agent.metadata.instanceUid", agent.Metadata.InstanceUID.String()),
		slog.String("agentgroup.metadata.name", agentGroup.Metadata.Name),
	)

	if agentGroup.HasAgentConnectionConfig() {
		logger.Debug("skip to apply connection settings because agentGroup has no connection config")

		return nil
	}

	conn := agentGroup.Spec.AgentConnectionConfig
	if conn == nil {
		return nil
	}

	opampConnection := s.buildOpAMPConnection(
		ctx, agentGroup.Metadata.Namespace, conn.OpAMPConnection, logger,
	)
	ownMetrics := s.buildTelemetryConnection(
		ctx, agentGroup.Metadata.Namespace, conn.OwnMetrics, logger,
	)
	ownLogs := s.buildTelemetryConnection(
		ctx, agentGroup.Metadata.Namespace, conn.OwnLogs, logger,
	)
	ownTraces := s.buildTelemetryConnection(
		ctx, agentGroup.Metadata.Namespace, conn.OwnTraces, logger,
	)
	otherConnections := s.buildOtherConnections(
		ctx, agentGroup.Metadata.Namespace, conn.OtherConnections, logger,
	)

	err := agent.ApplyConnectionSettings(opampConnection, ownMetrics, ownLogs, ownTraces, otherConnections)
	if err != nil {
		return fmt.Errorf("apply connection settings: %w", err)
	}

	return nil
}

func (s *AgentGroupService) buildOpAMPConnection(
	ctx context.Context,
	namespace string,
	conn *agentmodel.OpAMPConnectionSettings,
	logger *slog.Logger,
) *agentmodel.AgentOpAMPConnectionSettings {
	if conn == nil {
		return nil
	}

	result := &agentmodel.AgentOpAMPConnectionSettings{
		DestinationEndpoint: conn.DestinationEndpoint,
		Headers:             conn.Headers,
		Certificate:         nil,
	}

	if conn.CertificateName != nil {
		certificate, err := s.certificatePersistencePort.GetCertificate(ctx, namespace, *conn.CertificateName, nil)
		if err != nil {
			logger.Warn("failed to get certificate for OpAMP connection",
				slog.String("certificateName", *conn.CertificateName),
				slog.String("err", err.Error()),
			)
		} else {
			result.Certificate = certificate.ToAgentCertificate()
		}
	}

	return result
}

func (s *AgentGroupService) buildTelemetryConnection(
	ctx context.Context,
	namespace string,
	conn *agentmodel.TelemetryConnectionSettings,
	logger *slog.Logger,
) *agentmodel.AgentTelemetryConnectionSettings {
	if conn == nil {
		return nil
	}

	result := &agentmodel.AgentTelemetryConnectionSettings{
		DestinationEndpoint: conn.DestinationEndpoint,
		Headers:             conn.Headers,
		Certificate:         nil,
	}

	if conn.CertificateName != nil {
		certificate, err := s.certificatePersistencePort.GetCertificate(ctx, namespace, *conn.CertificateName, nil)
		if err != nil {
			logger.Warn("failed to get certificate for telemetry connection",
				slog.String("certificateName", *conn.CertificateName),
				slog.String("err", err.Error()),
			)

			return nil
		}

		result.Certificate = certificate.ToAgentCertificate()
	}

	return result
}

func (s *AgentGroupService) buildOtherConnections(
	ctx context.Context,
	namespace string,
	conns map[string]agentmodel.OtherConnectionSettings,
	logger *slog.Logger,
) map[string]agentmodel.AgentOtherConnectionSettings {
	return mapValuesWithFilterNil(conns,
		func(conn agentmodel.OtherConnectionSettings, _ string) *agentmodel.AgentOtherConnectionSettings {
			result := &agentmodel.AgentOtherConnectionSettings{
				DestinationEndpoint: conn.DestinationEndpoint,
				Headers:             conn.Headers,
				Certificate:         nil,
			}

			if conn.CertificateName != nil {
				certificate, err := s.certificatePersistencePort.GetCertificate(ctx, namespace, *conn.CertificateName, nil)
				if err != nil {
					logger.Warn("failed to get certificate for other connection",
						slog.String("certificateName", *conn.CertificateName),
						slog.String("err", err.Error()),
					)

					return nil
				}

				result.Certificate = certificate.ToAgentCertificate()
			}

			return result
		},
	)
}

func mapValuesWithFilterNil[K comparable, V, R any](in map[K]V, iteratee func(value V, key K) *R) map[K]R {
	result := make(map[K]R, len(in))

	for key, value := range in {
		transformed := iteratee(value, key)
		if transformed == nil {
			continue
		}

		result[key] = *transformed
	}

	return result
}
