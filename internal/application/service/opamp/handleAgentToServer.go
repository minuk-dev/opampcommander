package opamp

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	modelagent "github.com/minuk-dev/opampcommander/internal/domain/model/agent"
	"github.com/minuk-dev/opampcommander/internal/domain/model/remoteconfig"
)

// HandleAgentToServer handle a message from agent.
func (s *Service) HandleAgentToServer(ctx context.Context, agentToServer *protobufs.AgentToServer) error {
	instanceUID := uuid.UUID(agentToServer.GetInstanceUid())
	s.logger.Info("HandleAgentToServer",
		slog.String("instanceUID", instanceUID.String()),
		slog.String("message", "start"),
	)

	agent, err := s.agentUsecase.GetOrCreateAgent(ctx, instanceUID)
	if err != nil {
		return fmt.Errorf("failed to get or create agent: %w", err)
	}

	s.logger.Debug("HandleAgentToServer",
		slog.String("instanceUID", instanceUID.String()),
		slog.String("message", "get or create agent"),
		slog.Any("agent", agent),
	)

	err = s.report(agent, agentToServer)
	if err != nil {
		return fmt.Errorf("failed to report: %w", err)
	}

	s.logger.Debug("HandleAgentToServer",
		slog.String("instanceUID", instanceUID.String()),
		slog.String("message", "report"),
		slog.Any("agent", agent),
	)

	err = s.agentUsecase.SaveAgent(ctx, agent)
	if err != nil {
		return fmt.Errorf("failed to save agent: %w", err)
	}

	s.logger.Info("HandleAgentToServer",
		slog.String("instanceUID", instanceUID.String()),
		slog.String("message", "success"),
	)

	_, err = s.connectionUsecase.GetOrCreateConnection(instanceUID)
	if err != nil {
		return fmt.Errorf("failed to get or create connection: %w", err)
	}

	return nil
}

func descToDomain(desc *protobufs.AgentDescription) *modelagent.Description {
	if desc == nil {
		return nil
	}

	return &modelagent.Description{
		IdentifyingAttributes:    toMap(desc.GetIdentifyingAttributes()),
		NonIdentifyingAttributes: toMap(desc.GetNonIdentifyingAttributes()),
	}
}

func (s *Service) report(agent *model.Agent, agentToServer *protobufs.AgentToServer) error {
	err := agent.ReportDescription(descToDomain(agentToServer.GetAgentDescription()))
	if err != nil {
		return fmt.Errorf("failed to report description: %w", err)
	}

	err = agent.ReportComponentHealth(healthToDomain(agentToServer.GetHealth()))
	if err != nil {
		return fmt.Errorf("failed to report component health: %w", err)
	}

	err = agent.ReportEffectiveConfig(effectiveConfigToDomain(agentToServer.GetEffectiveConfig()))
	if err != nil {
		return fmt.Errorf("failed to report effective config: %w", err)
	}

	err = agent.ReportRemoteConfigStatus(remoteConfigStatusToDomain(agentToServer.GetRemoteConfigStatus()))
	if err != nil {
		return fmt.Errorf("failed to report remote config status: %w", err)
	}

	err = agent.ReportPackageStatuses(packageStatusToDomain(agentToServer.GetPackageStatuses()))
	if err != nil {
		return fmt.Errorf("failed to report package statuses: %w", err)
	}

	err = agent.ReportCustomCapabilities(customCapabilitiesToDomain(agentToServer.GetCustomCapabilities()))
	if err != nil {
		return fmt.Errorf("failed to report custom capabilities: %w", err)
	}

	err = agent.ReportAvailableComponents(availableComponentsToDomain(agentToServer.GetAvailableComponents()))
	if err != nil {
		return fmt.Errorf("failed to report available components: %w", err)
	}

	return nil
}

func remoteConfigStatusToDomain(remoteConfigStatus *protobufs.RemoteConfigStatus) *model.AgentRemoteConfigStatus {
	if remoteConfigStatus == nil {
		return nil
	}

	return &model.AgentRemoteConfigStatus{
		LastRemoteConfigHash: remoteConfigStatus.GetLastRemoteConfigHash(),
		Status:               remoteconfig.Status(remoteConfigStatus.GetStatus()),
		ErrorMessage:         remoteConfigStatus.GetErrorMessage(),
	}
}

func customCapabilitiesToDomain(customCapabilities *protobufs.CustomCapabilities) *model.AgentCustomCapabilities {
	if customCapabilities == nil {
		return nil
	}

	return &model.AgentCustomCapabilities{
		Capabilities: customCapabilities.GetCapabilities(),
	}
}

func toMap(proto []*protobufs.KeyValue) map[string]string {
	retval := make(map[string]string, len(proto))
	for _, kv := range proto {
		// iss#1: Handle other types.
		retval[kv.GetKey()] = kv.GetValue().GetStringValue()
	}

	return retval
}

func healthToDomain(health *protobufs.ComponentHealth) *model.AgentComponentHealth {
	if health == nil {
		return nil
	}

	componentHealthMap := make(map[string]model.AgentComponentHealth, len(health.GetComponentHealthMap()))

	for subComponentName, subComponentHealth := range health.GetComponentHealthMap() {
		componentHealthMap[subComponentName] = *healthToDomain(subComponentHealth)
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
