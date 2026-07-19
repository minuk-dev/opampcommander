package opamp

import (
	"encoding/base64"
	"strconv"
	"time"

	"github.com/open-telemetry/opamp-go/protobufs"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	modelagent "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/agent"
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

func remoteConfigStatusToDomain(remoteConfigStatus *protobufs.RemoteConfigStatus) *agentmodel.AgentRemoteConfigStatus {
	if remoteConfigStatus == nil {
		return nil
	}

	return &agentmodel.AgentRemoteConfigStatus{
		LastRemoteConfigHash: remoteConfigStatus.GetLastRemoteConfigHash(),
		Status:               agentmodel.RemoteConfigStatus(remoteConfigStatus.GetStatus()),
		ErrorMessage:         remoteConfigStatus.GetErrorMessage(),
		LastUpdatedAt:        time.Now(),
	}
}

func connectionSettingsStatusToDomain(
	connectionSettingsStatus *protobufs.ConnectionSettingsStatus,
) *agentmodel.AgentConnectionSettingsStatus {
	if connectionSettingsStatus == nil {
		return nil
	}

	return &agentmodel.AgentConnectionSettingsStatus{
		LastConnectionSettingsHash: connectionSettingsStatus.GetLastConnectionSettingsHash(),
		Status:                     agentmodel.ConnectionSettingsStatus(connectionSettingsStatus.GetStatus()),
		ErrorMessage:               connectionSettingsStatus.GetErrorMessage(),
	}
}

func customCapabilitiesToDomain(customCapabilities *protobufs.CustomCapabilities) *agentmodel.AgentCustomCapabilities {
	if customCapabilities == nil {
		return nil
	}

	return &agentmodel.AgentCustomCapabilities{
		Capabilities: customCapabilities.GetCapabilities(),
	}
}

func toMap(proto []*protobufs.KeyValue) map[string]string {
	retval := make(map[string]string, len(proto))
	for _, kv := range proto {
		retval[kv.GetKey()] = anyValueToString(kv.GetValue())
	}

	return retval
}

// anyValueToString renders an OpAMP AnyValue as a string for storage in the agent's
// string-keyed attribute maps. Scalar values keep their natural representation (so a
// non-string identifying attribute such as an int process.pid is preserved instead of being
// silently dropped to ""); bytes are base64-encoded and nested array/kvlist values fall back
// to their protobuf text form.
func anyValueToString(value *protobufs.AnyValue) string {
	if value == nil {
		return ""
	}

	switch value.GetValue().(type) {
	case *protobufs.AnyValue_StringValue:
		return value.GetStringValue()
	case *protobufs.AnyValue_BoolValue:
		return strconv.FormatBool(value.GetBoolValue())
	case *protobufs.AnyValue_IntValue:
		return strconv.FormatInt(value.GetIntValue(), 10)
	case *protobufs.AnyValue_DoubleValue:
		return strconv.FormatFloat(value.GetDoubleValue(), 'g', -1, 64)
	case *protobufs.AnyValue_BytesValue:
		return base64.StdEncoding.EncodeToString(value.GetBytesValue())
	case *protobufs.AnyValue_ArrayValue, *protobufs.AnyValue_KvlistValue:
		return value.String()
	default:
		return ""
	}
}

func healthToDomain(health *protobufs.ComponentHealth) *agentmodel.AgentComponentHealth {
	if health == nil {
		return nil
	}

	componentHealthMap := make(map[string]agentmodel.AgentComponentHealth, len(health.GetComponentHealthMap()))

	for subComponentName, subComponentHealth := range health.GetComponentHealthMap() {
		if converted := healthToDomain(subComponentHealth); converted != nil {
			componentHealthMap[subComponentName] = *converted
		}
	}

	return &agentmodel.AgentComponentHealth{
		Healthy:            health.GetHealthy(),
		StartTime:          timeutil.UnixNanoToTime(health.GetStartTimeUnixNano()),
		LastError:          health.GetLastError(),
		Status:             health.GetStatus(),
		StatusTime:         timeutil.UnixNanoToTime(health.GetStatusTimeUnixNano()),
		ComponentHealthMap: componentHealthMap,
	}
}

func effectiveConfigToDomain(effectiveConfig *protobufs.EffectiveConfig) *agentmodel.AgentEffectiveConfig {
	if effectiveConfig == nil {
		return nil
	}

	configMap := make(map[string]agentmodel.AgentConfigFile, len(effectiveConfig.GetConfigMap().GetConfigMap()))
	for key, value := range effectiveConfig.GetConfigMap().GetConfigMap() {
		configMap[key] = agentmodel.AgentConfigFile{
			Body:        value.GetBody(),
			ContentType: value.GetContentType(),
		}
	}

	return &agentmodel.AgentEffectiveConfig{
		ConfigMap: agentmodel.AgentConfigMap{
			ConfigMap: configMap,
		},
	}
}

func packageStatusToDomain(packageStatuses *protobufs.PackageStatuses) *agentmodel.AgentPackageStatuses {
	if packageStatuses == nil {
		return nil
	}

	packages := make(map[string]agentmodel.AgentPackageStatusEntry, len(packageStatuses.GetPackages()))
	for key, value := range packageStatuses.GetPackages() {
		packages[key] = agentmodel.AgentPackageStatusEntry{
			Name:                 value.GetName(),
			AgentHasVersion:      value.GetAgentHasVersion(),
			AgentHasHash:         value.GetAgentHasHash(),
			ServerOfferedVersion: value.GetServerOfferedVersion(),
			Status:               agentmodel.AgentPackageStatusEnum(value.GetStatus()),
			ErrorMessage:         value.GetErrorMessage(),
		}
	}

	return &agentmodel.AgentPackageStatuses{
		Packages:                     packages,
		ServerProvidedAllPackgesHash: packageStatuses.GetServerProvidedAllPackagesHash(),
		ErrorMessage:                 packageStatuses.GetErrorMessage(),
	}
}

func availableComponentsToDomain(
	availableComponents *protobufs.AvailableComponents,
) *agentmodel.AgentAvailableComponents {
	if availableComponents == nil {
		return nil
	}

	components := make(map[string]agentmodel.ComponentDetails, len(availableComponents.GetComponents()))
	for key, value := range availableComponents.GetComponents() {
		components[key] = componentDetailsToDomain(value)
	}

	return &agentmodel.AgentAvailableComponents{
		Components: components,
		Hash:       availableComponents.GetHash(),
	}
}

func componentDetailsToDomain(componentDetails *protobufs.ComponentDetails) agentmodel.ComponentDetails {
	metadata := toMap(componentDetails.GetMetadata())

	subComponentMap := make(map[string]agentmodel.ComponentDetails, len(componentDetails.GetSubComponentMap()))
	for key, value := range componentDetails.GetSubComponentMap() {
		subComponentMap[key] = componentDetailsToDomain(value)
	}

	return agentmodel.ComponentDetails{
		Metadata:        metadata,
		SubComponentMap: subComponentMap,
	}
}
