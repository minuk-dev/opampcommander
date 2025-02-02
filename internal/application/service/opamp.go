package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"

	applicationport "github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ applicationport.OpAMPUsecase = (*OpAMPService)(nil)

type OpAMPService struct {
	logger            *slog.Logger
	connectionUsecase domainport.ConnectionUsecase
	agentUsecase      domainport.AgentUsecase
}

func NewOpAMPService(
	connectionUsecase domainport.ConnectionUsecase,
	agentUsecase domainport.AgentUsecase,
	logger *slog.Logger,
) *OpAMPService {
	return &OpAMPService{
		logger:            logger,
		connectionUsecase: connectionUsecase,
		agentUsecase:      agentUsecase,
	}
}

// HandleAgentToServer handle a message from agent.
func (s *OpAMPService) HandleAgentToServer(ctx context.Context, agentToServer *protobufs.AgentToServer) error {
	instanceUID := uuid.UUID(agentToServer.GetInstanceUid())
	s.logger.Info("HandleAgentToServer",
		slog.String("instanceUID", instanceUID.String()),
		slog.String("message", "start"),
	)

	agent, err := s.agentUsecase.GetOrCreateAgent(ctx, instanceUID)
	if err != nil {
		return fmt.Errorf("failed to get or create agent: %w", err)
	}

	err = s.report(agent, agentToServer)
	if err != nil {
		return fmt.Errorf("failed to report: %w", err)
	}

	err = s.agentUsecase.SaveAgent(ctx, agent)
	if err != nil {
		return fmt.Errorf("failed to save agent: %w", err)
	}

	s.logger.Info("HandleAgentToServer",
		slog.String("instanceUID", instanceUID.String()),
		slog.String("message", "success"),
	)

	conn, err := s.connectionUsecase.GetOrCreateConnection(instanceUID)
	if err != nil {
		return fmt.Errorf("failed to get or create connection: %w", err)
	}

	// iss#2: Make more proper serverToAgent from agent.
	//exhaustruct:ignore
	serverToAgent := &protobufs.ServerToAgent{
		InstanceUid: agentToServer.GetInstanceUid(),
	}

	err = conn.SendServerToAgent(ctx, serverToAgent)
	if err != nil {
		return fmt.Errorf("failed to send a message to the channel: %w", err)
	}

	return nil
}

// FetchServerToAgent fetch a message.
func (s *OpAMPService) FetchServerToAgent(
	ctx context.Context,
	instanceUID uuid.UUID,
) (*protobufs.ServerToAgent, error) {
	conn, err := s.connectionUsecase.GetConnection(instanceUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	serverToAgent, err := conn.FetchServerToAgent(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch a message from the channel: %w", err)
	}

	return serverToAgent, nil
}

func (s *OpAMPService) DisconnectAgent(instanceUID uuid.UUID) error {
	conn, err := s.connectionUsecase.FetchAndDeleteConnection(instanceUID)
	if err != nil && errors.Is(err, domainport.ErrConnectionNotFound) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to delete connection: %w", err)
	}

	if conn != nil {
		return fmt.Errorf("connection is nil: %w", err)
	}

	err = conn.Close()
	if err != nil {
		return fmt.Errorf("failed to close connection: %w", err)
	}

	return nil
}

func (s *OpAMPService) report(agent *model.Agent, agentToServer *protobufs.AgentToServer) error {
	desc := &model.AgentDescription{
		IdentifyingAttributes:    toMap(agentToServer.GetAgentDescription().GetIdentifyingAttributes()),
		NonIdentifyingAttributes: toMap(agentToServer.GetAgentDescription().GetNonIdentifyingAttributes()),
	}

	err := agent.ReportDescription(desc)
	if err != nil {
		return fmt.Errorf("failed to report description: %w", err)
	}

	err = agent.ReportComponentHealth(toDomain(agentToServer.GetHealth()))
	if err != nil {
		return fmt.Errorf("failed to report component health: %w", err)
	}

	err = agent.ReportEffectiveConfig(effectiveConfigToDomain(agentToServer.GetEffectiveConfig()))
	if err != nil {
		return fmt.Errorf("failed to report effective config: %w", err)
	}

	err = agent.ReportRemoteConfigStatus(&model.AgentRemoteConfigStatus{
		LastRemoteConfigHash: agentToServer.GetRemoteConfigStatus().GetLastRemoteConfigHash(),
		Status:               model.AgentRemoteConfigStatusEnum(agentToServer.GetRemoteConfigStatus().GetStatus()),
		ErrorMessage:         agentToServer.GetRemoteConfigStatus().GetErrorMessage(),
	})
	if err != nil {
		return fmt.Errorf("failed to report remote config status: %w", err)
	}

	err = agent.ReportPackageStatuses(packageStatusToDomain(agentToServer.GetPackageStatuses()))
	if err != nil {
		return fmt.Errorf("failed to report package statuses: %w", err)
	}

	err = agent.ReportCustomCapabilities(&model.AgentCustomCapabilities{
		Capabilities: agentToServer.GetCustomCapabilities().GetCapabilities(),
	})
	if err != nil {
		return fmt.Errorf("failed to custom capabilities: %w", err)
	}

	err = agent.ReportAvailableComponents(availableComponentsToDomain(agentToServer.GetAvailableComponents()))
	if err != nil {
		return fmt.Errorf("failed to report available components: %w", err)
	}

	return nil
}

func toMap(proto []*protobufs.KeyValue) map[string]string {
	retval := make(map[string]string, len(proto))
	for _, kv := range proto {
		// iss#1: Handle other types.
		retval[kv.GetKey()] = kv.GetValue().GetStringValue()
	}

	return retval
}

func toDomain(health *protobufs.ComponentHealth) *model.AgentComponentHealth {
	componentHealthMap := make(map[string]model.AgentComponentHealth, len(health.GetComponentHealthMap()))

	for subComponentName, subComponentHealth := range health.GetComponentHealthMap() {
		componentHealthMap[subComponentName] = *toDomain(subComponentHealth)
	}

	return &model.AgentComponentHealth{
		Healthy:            health.GetHealthy(),
		StartTime:          unixNanoToTime(health.GetStartTimeUnixNano()),
		LastError:          health.GetLastError(),
		Status:             health.GetStatus(),
		StatusTime:         unixNanoToTime(health.GetStatusTimeUnixNano()),
		ComponentHealthMap: componentHealthMap,
	}
}

//nolint:mnd,gosec
func unixNanoToTime(nsec uint64) time.Time {
	sec := nsec / 1e9
	nsec %= 1e9

	return time.Unix(int64(sec), int64(nsec))
}

func effectiveConfigToDomain(effectiveConfig *protobufs.EffectiveConfig) *model.AgentEffectiveConfig {
	configMap := make(map[string]model.AgentConfigFile, len(effectiveConfig.GetConfigMap().GetConfigMap()))
	for key, value := range effectiveConfig.GetConfigMap().GetConfigMap() {
		configMap[key] = model.AgentConfigFile{
			Body:        value.GetBody(),
			ContentType: value.GetContentType(),
		}
	}

	return &model.AgentEffectiveConfig{
		ConfigMap: model.AgentConfigMap{
			ConfigMap: configMap,
		},
	}
}

func packageStatusToDomain(packageStatuses *protobufs.PackageStatuses) *model.AgentPackageStatuses {
	packages := make(map[string]model.AgentPackageStatus, len(packageStatuses.GetPackages()))
	for key, value := range packageStatuses.GetPackages() {
		packages[key] = model.AgentPackageStatus{
			Name:                 value.GetName(),
			AgentHasVersion:      value.GetAgentHasVersion(),
			AgentHasHash:         value.GetAgentHasHash(),
			ServerOfferedVersion: value.GetServerOfferedVersion(),
			Status:               model.AgentPackageStatusEnum(value.GetStatus()),
			ErrorMessage:         value.GetErrorMessage(),
		}
	}

	return &model.AgentPackageStatuses{
		Packages:                     packages,
		ServerProvidedAllPackgesHash: packageStatuses.GetServerProvidedAllPackagesHash(),
		ErrorMessage:                 packageStatuses.GetErrorMessage(),
	}
}

func availableComponentsToDomain(availableComponents *protobufs.AvailableComponents) *model.AgentAvailableComponents {
	components := make(map[string]model.ComponentDetails, len(availableComponents.GetComponents()))
	for key, value := range availableComponents.GetComponents() {
		components[key] = componentDetailsToDomain(value)
	}

	return &model.AgentAvailableComponents{
		Components: components,
		Hash:       availableComponents.GetHash(),
	}
}

func componentDetailsToDomain(componentDetails *protobufs.ComponentDetails) model.ComponentDetails {
	metadata := toMap(componentDetails.GetMetadata())

	subComponentMap := make(map[string]model.ComponentDetails, len(componentDetails.GetSubComponentMap()))
	for key, value := range componentDetails.GetSubComponentMap() {
		subComponentMap[key] = componentDetailsToDomain(value)
	}

	return model.ComponentDetails{
		Metadata:        metadata,
		SubComponentMap: subComponentMap,
	}
}
