package opamp

import (
	"time"

	"github.com/open-telemetry/opamp-go/protobufs"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	modelagent "github.com/minuk-dev/opampcommander/internal/domain/model/agent"
	"github.com/minuk-dev/opampcommander/pkg/timeutil"
)

func descToDomain(desc *protobufs.AgentDescription) *modelagent.Description {
	if desc == nil {
		return nil
	}

	return &modelagent.Description{
		IdentifyingAttributes:    toMap(desc.GetIdentifyingAttributes()),
		NonIdentifyingAttributes: toMap(desc.GetNonIdentifyingAttributes()),
	}
}

func remoteConfigStatusToDomain(remoteConfigStatus *protobufs.RemoteConfigStatus) *model.AgentRemoteConfigStatus {
	if remoteConfigStatus == nil {
		return nil
	}

	return &model.AgentRemoteConfigStatus{
		LastRemoteConfigHash: remoteConfigStatus.GetLastRemoteConfigHash(),
		Status:               model.RemoteConfigStatus(remoteConfigStatus.GetStatus()),
		ErrorMessage:         remoteConfigStatus.GetErrorMessage(),
		LastUpdatedAt:        time.Now(),
	}
}

func connectionSettingsStatusToDomain(
	connectionSettingsStatus *protobufs.ConnectionSettingsStatus,
) *model.AgentConnectionSettingsStatus {
	if connectionSettingsStatus == nil {
		return nil
	}

	return &model.AgentConnectionSettingsStatus{
		LastConnectionSettingsHash: connectionSettingsStatus.GetLastConnectionSettingsHash(),
		Status:                     model.ConnectionSettingsStatus(connectionSettingsStatus.GetStatus()),
		ErrorMessage:               connectionSettingsStatus.GetErrorMessage(),
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
		StartTime:          timeutil.UnixNanoToTime(health.GetStartTimeUnixNano()),
		LastError:          health.GetLastError(),
		Status:             health.GetStatus(),
		StatusTime:         timeutil.UnixNanoToTime(health.GetStatusTimeUnixNano()),
		ComponentHealthMap: componentHealthMap,
	}
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
	packages := make(map[string]model.AgentPackageStatusEntry, len(packageStatuses.GetPackages()))
	for key, value := range packageStatuses.GetPackages() {
		packages[key] = model.AgentPackageStatusEntry{
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
