package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	domainmodel "github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/agent"
)

const (
	// AgentKeyFieldName is the field name used as the key for Agent entities in MongoDB.
	AgentKeyFieldName string = "metadata.instanceUid"

	// IdentifyingAttributesFieldName is the field name for identifying attributes in MongoDB.
	// It is indexed for efficient querying.
	IdentifyingAttributesFieldName string = "metadata.description.identifyingAttributes"
	// NonIdentifyingAttributesFieldName is the field name for non-identifying attributes in MongoDB.
	// It is indexed for efficient querying.
	NonIdentifyingAttributesFieldName string = "metadata.description.nonIdentifyingAttributes"
)

// Agent is a struct that represents the MongoDB entity for an Agent.
type Agent struct {
	Common `bson:",inline"`

	Metadata AgentMetadata `bson:"metadata"`
	Spec     AgentSpec     `bson:"spec"`
	Status   AgentStatus   `bson:"status"`
}

// AgentMetadata represents the metadata of an agent.
type AgentMetadata struct {
	InstanceUID        bson.Binary              `bson:"instanceUid"`
	InstanceUIDString  string                   `bson:"instanceUidString"` // String representation for searching
	Capabilities       *AgentCapabilities       `bson:"capabilities,omitempty"`
	Description        *AgentDescription        `bson:"description,omitempty"`
	CustomCapabilities *AgentCustomCapabilities `bson:"customCapabilities,omitempty"`
}

// AgentSpec represents the desired specification of an agent.
type AgentSpec struct {
	NewInstanceUID      *bson.Binary           `bson:"newInstanceUID,omitempty"`
	RemoteConfig        *AgentSpecRemoteConfig `bson:"remoteConfig,omitempty"`
	RequiredRestartedAt bson.DateTime          `bson:"requiredRestartedAt,omitempty"`
}

// AgentStatus represents the current status of an agent.
type AgentStatus struct {
	EffectiveConfig     *AgentEffectiveConfig     `bson:"effectiveConfig,omitempty"`
	PackageStatuses     *AgentPackageStatuses     `bson:"packageStatuses,omitempty"`
	ComponentHealth     *AgentComponentHealth     `bson:"componentHealth,omitempty"`
	AvailableComponents *AgentAvailableComponents `bson:"availableComponents,omitempty"`
	RemoteConfigStatus  *AgentRemoteConfigStatus  `bson:"remoteConfigStatus,omitempty"`
	// Conditions stores agent conditions for informational purposes only.
	// WARNING: Do NOT use Conditions for MongoDB queries or aggregations.
	// The Conditions field can be null which causes MongoDB aggregation errors.
	// Use the following indexed fields instead:
	// - Connected (bool): for connection status queries
	// - ComponentHealth.Healthy (bool): for health status queries
	Conditions         []AgentCondition `bson:"conditions,omitempty"`
	Connected          bool             `bson:"connected,omitempty"`
	ConnectionType     string           `bson:"connectionType,omitempty"`
	SequenceNum        uint64           `bson:"sequenceNum,omitempty"`
	LastCommunicatedAt bson.DateTime    `bson:"lastCommunicatedAt,omitempty"`
	LastCommunicatedTo string           `bson:"lastCommunicatedTo,omitempty"`
}

// AgentCondition represents a condition of an agent in MongoDB.
type AgentCondition struct {
	Type               string        `bson:"type"`
	LastTransitionTime bson.DateTime `bson:"lastTransitionTime"`
	Status             string        `bson:"status"`
	Reason             string        `bson:"reason"`
	Message            string        `bson:"message,omitempty"`
}

// AgentDescription is a struct to manage agent description.
type AgentDescription struct {
	IdentifyingAttributes    KeyValuePairs `bson:"identifyingAttributes,omitempty"`
	NonIdentifyingAttributes KeyValuePairs `bson:"nonIdentifyingAttributes,omitempty"`
}

// KeyValuePair is a struct to manage key-value pairs.
type KeyValuePair struct {
	Key   string `bson:"key"`
	Value string `bson:"value"`
}

// KeyValuePairs is a slice of KeyValuePair.
type KeyValuePairs []KeyValuePair

// ToMap converts KeyValuePairs to a map.
func (kvs KeyValuePairs) ToMap() map[string]string {
	m := make(map[string]string, len(kvs))
	for _, kv := range kvs {
		m[kv.Key] = kv.Value
	}

	return m
}

// MapToKeyValuePairs converts a map to KeyValuePairs.
func MapToKeyValuePairs(input map[string]string) KeyValuePairs {
	if input == nil {
		return nil
	}

	kvs := make(KeyValuePairs, 0, len(input))
	for k, v := range input {
		kvs = append(kvs, KeyValuePair{
			Key:   k,
			Value: v,
		})
	}

	return kvs
}

// AgentComponentHealth is a struct to manage component health.
type AgentComponentHealth struct {
	Healthy             bool                            `bson:"healthy"`
	StartTimeUnixMilli  int64                           `bson:"startTimeUnixMilli"`
	LastError           string                          `bson:"lastError"`
	Status              string                          `bson:"status"`
	StatusTimeUnixMilli int64                           `bson:"statusTimeUnixMilli"`
	ComponentHealthMap  map[string]AgentComponentHealth `bson:"componentHealthMap,omitempty"`
}

// AgentCapabilities is a bitmask of capabilities that the Agent supports.
type AgentCapabilities uint64

// AgentEffectiveConfig is a struct to manage effective config.
type AgentEffectiveConfig struct {
	ConfigMap AgentConfigMap `bson:"configMap"`
}

// AgentConfigMap is a struct to manage config map.
type AgentConfigMap struct {
	ConfigMap map[string]AgentConfigFile `bson:"configMap,omitempty"`
}

// AgentConfigFile is a struct to manage config file.
type AgentConfigFile struct {
	Body        bson.Binary `bson:"body"`
	ContentType string      `bson:"contentType"`
}

// AgentSpecRemoteConfig is a struct to manage remote config names for agent spec.
type AgentSpecRemoteConfig struct {
	RemoteConfig []string `bson:"remoteConfig"`
}

// AgentRemoteConfigData is a struct to manage remote config data.
type AgentRemoteConfigData struct {
	Key                bson.Binary                 `bson:"key"`
	Status             AgentRemoteConfigStatusEnum `bson:"status"`
	Config             bson.Binary                 `bson:"config"`
	LastUpdatedAtMilli int64                       `bson:"lastUpdatedAtMilli"`
}

// AgentRemoteConfigSub is a struct to manage remote config status with key.
type AgentRemoteConfigSub struct {
	Key   bson.Binary                 `bson:"key"`
	Value AgentRemoteConfigStatusEnum `bson:"value"`
}

// AgentRemoteConfigStatusEnum is an enum that represents the status of the remote config.
type AgentRemoteConfigStatusEnum int32

// AgentRemoteConfigStatus represents the status of remote config in MongoDB.
type AgentRemoteConfigStatus struct {
	LastRemoteConfigHash *bson.Binary                `bson:"lastRemoteConfigHash,omitempty"`
	Status               AgentRemoteConfigStatusEnum `bson:"status"`
	ErrorMessage         string                      `bson:"errorMessage,omitempty"`
}

// AgentPackageStatuses is a map of package statuses.
type AgentPackageStatuses struct {
	Packages                     map[string]AgentPackageStatus `bson:"packages"`
	ServerProvidedAllPackgesHash bson.Binary                   `bson:"serverProvidedAllPackgesHash"`
	ErrorMessage                 string                        `bson:"errorMessage"`
}

// AgentPackageStatus is a status of a package.
type AgentPackageStatus struct {
	Name                 string                 `bson:"name"`
	AgentHasVersion      string                 `bson:"agentHasVersion"`
	AgentHasHash         bson.Binary            `bson:"agentHasHash"`
	ServerOfferedVersion string                 `bson:"serverOfferedVersion"`
	Status               AgentPackageStatusEnum `bson:"status"`
	ErrorMessage         string                 `bson:"errorMessage"`
}

// AgentPackageStatusEnum is an enum that represents the status of a package.
type AgentPackageStatusEnum int32

// AgentCustomCapabilities is a custom capabilities of the agent.
type AgentCustomCapabilities struct {
	Capabilities []string `bson:"capabilities"`
}

// AgentAvailableComponents is a map of available components.
type AgentAvailableComponents struct {
	Components map[string]ComponentDetails `bson:"components"`
	Hash       bson.Binary                 `bson:"hash"`
}

// ComponentDetails is a details of a component.
type ComponentDetails struct {
	Metadata        map[string]string           `bson:"metadata"`
	SubComponentMap map[string]ComponentDetails `bson:"subComponentMap"`
}

// ToDomain converts the Agent to domain model.
func (a *Agent) ToDomain() *domainmodel.Agent {
	return &domainmodel.Agent{
		Metadata: a.Metadata.ToDmain(),
		Spec:     a.Spec.ToDomain(),
		Status:   a.Status.ToDomain(),
	}
}

// ToDmain converts the AgentMetadata to domain model.
func (metadata *AgentMetadata) ToDmain() domainmodel.AgentMetadata {
	return domainmodel.AgentMetadata{
		InstanceUID: uuid.UUID(metadata.InstanceUID.Data),
		Description: switchIfNil(
			metadata.Description.ToDomain(),
			//exhaustruct:ignore
			agent.Description{},
		),
		Capabilities: switchIfNil(
			metadata.Capabilities.ToDomain(),
			agent.Capabilities(0),
		),
		CustomCapabilities: switchIfNil(
			metadata.CustomCapabilities.ToDomain(),
			//exhaustruct:ignore
			domainmodel.AgentCustomCapabilities{},
		),
	}
}

// ToDomain converts the AgentSpec to domain model.
func (spec *AgentSpec) ToDomain() domainmodel.AgentSpec {
	var uid uuid.UUID

	const uuidSize = 16

	if spec.NewInstanceUID != nil && len(spec.NewInstanceUID.Data) == uuidSize {
		copy(uid[:], spec.NewInstanceUID.Data)
	}

	//exhaustruct:ignore
	agentSpec := domainmodel.AgentSpec{}
	agentSpec.NewInstanceUID = uid
	agentSpec.RestartInfo = &domainmodel.AgentRestartInfo{
		RequiredRestartedAt: time.Time{},
	}
	agentSpec.ConnectionInfo = nil
	agentSpec.RemoteConfig = spec.RemoteConfig.ToDomainPtr()

	return agentSpec
}

// ToDomain converts the AgentStatus to domain model.
func (status *AgentStatus) ToDomain() domainmodel.AgentStatus {
	conditions := make([]domainmodel.AgentCondition, len(status.Conditions))
	for i, condition := range status.Conditions {
		conditions[i] = domainmodel.AgentCondition{
			Type:               domainmodel.AgentConditionType(condition.Type),
			LastTransitionTime: condition.LastTransitionTime.Time(),
			Status:             domainmodel.AgentConditionStatus(condition.Status),
			Reason:             condition.Reason,
			Message:            condition.Message,
		}
	}

	//exhaustruct:ignore
	return domainmodel.AgentStatus{
		EffectiveConfig: switchIfNil(
			status.EffectiveConfig.ToDomain(),
			//exhaustruct:ignore
			domainmodel.AgentEffectiveConfig{},
		),
		PackageStatuses: switchIfNil(
			status.PackageStatuses.ToDomain(),
			//exhaustruct:ignore
			domainmodel.AgentPackageStatuses{},
		),
		ComponentHealth: switchIfNil(
			status.ComponentHealth.ToDomain(),
			//exhaustruct:ignore
			domainmodel.AgentComponentHealth{},
		),
		AvailableComponents: switchIfNil(
			status.AvailableComponents.ToDomain(),
			//exhaustruct:ignore
			domainmodel.AgentAvailableComponents{},
		),
		RemoteConfigStatus: func() domainmodel.AgentRemoteConfigStatus {
			if status.RemoteConfigStatus == nil {
				return domainmodel.AgentRemoteConfigStatus{
					LastRemoteConfigHash: nil,
					Status:               domainmodel.RemoteConfigStatusUnset,
					ErrorMessage:         "",
					LastUpdatedAt:        time.Time{},
				}
			}

			return status.RemoteConfigStatus.ToDomain()
		}(),
		ConnectionSettingsStatus: domainmodel.AgentConnectionSettingsStatus{
			LastConnectionSettingsHash: nil,
			Status:                     domainmodel.ConnectionSettingsStatusUnset,
			ErrorMessage:               "",
		},
		Conditions:     conditions,
		Connected:      status.Connected,
		ConnectionType: domainmodel.ConnectionTypeFromString(status.ConnectionType),
		SequenceNum:    status.SequenceNum,
		LastReportedAt: status.LastCommunicatedAt.Time(),
		LastReportedTo: status.LastCommunicatedTo,
	}
}

// ToDomain converts the AgentCapabilities to domain model.
func (ac *AgentCapabilities) ToDomain() *agent.Capabilities {
	if ac == nil {
		return nil
	}

	return (*agent.Capabilities)(ac)
}

// ToDomain converts the AgentDescription to domain model.
func (ad *AgentDescription) ToDomain() *agent.Description {
	if ad == nil {
		return nil
	}

	return &agent.Description{
		IdentifyingAttributes:    ad.IdentifyingAttributes.ToMap(),
		NonIdentifyingAttributes: ad.NonIdentifyingAttributes.ToMap(),
	}
}

// ToDomain converts the AgentEffectiveConfig to domain model.
func (ae *AgentEffectiveConfig) ToDomain() *domainmodel.AgentEffectiveConfig {
	if ae == nil {
		return nil
	}

	return &domainmodel.AgentEffectiveConfig{
		ConfigMap: domainmodel.AgentConfigMap{
			ConfigMap: lo.MapValues(ae.ConfigMap.ConfigMap, func(acf AgentConfigFile, _ string) domainmodel.AgentConfigFile {
				return domainmodel.AgentConfigFile{
					Body:        acf.Body.Data,
					ContentType: acf.ContentType,
				}
			}),
		},
	}
}

// ToDomain converts the AgentPackageStatuses to domain model.
func (ap *AgentPackageStatuses) ToDomain() *domainmodel.AgentPackageStatuses {
	if ap == nil {
		return nil
	}

	return &domainmodel.AgentPackageStatuses{
		Packages: lo.MapValues(ap.Packages, func(aps AgentPackageStatus, _ string) domainmodel.AgentPackageStatusEntry {
			return domainmodel.AgentPackageStatusEntry{
				Name:                 aps.Name,
				AgentHasVersion:      aps.AgentHasVersion,
				AgentHasHash:         aps.AgentHasHash.Data,
				ServerOfferedVersion: aps.ServerOfferedVersion,
				Status:               domainmodel.AgentPackageStatusEnum(aps.Status),
				ErrorMessage:         aps.ErrorMessage,
			}
		}),
		ServerProvidedAllPackgesHash: ap.ServerProvidedAllPackgesHash.Data,
		ErrorMessage:                 ap.ErrorMessage,
	}
}

// ToDomain converts the AgentComponentHealth to domain model.
func (ach *AgentComponentHealth) ToDomain() *domainmodel.AgentComponentHealth {
	if ach == nil {
		return nil
	}

	return &domainmodel.AgentComponentHealth{
		Healthy:    ach.Healthy,
		StartTime:  time.UnixMilli(ach.StartTimeUnixMilli),
		LastError:  ach.LastError,
		Status:     ach.Status,
		StatusTime: time.UnixMilli(ach.StatusTimeUnixMilli),
		ComponentHealthMap: lo.MapValues(ach.ComponentHealthMap,
			func(ach AgentComponentHealth, _ string) domainmodel.AgentComponentHealth {
				return *ach.ToDomain()
			}),
	}
}

// ToDomain converts the AgentSpecRemoteConfig to domain model.
func (asrc *AgentSpecRemoteConfig) ToDomain() domainmodel.AgentSpecRemoteConfig {
	if asrc == nil || len(asrc.RemoteConfig) == 0 {
		return domainmodel.AgentSpecRemoteConfig{
			ConfigMap: domainmodel.AgentConfigMap{
				ConfigMap: nil,
			},
		}
	}

	// Convert RemoteConfig names to ConfigMap with empty placeholders
	configMap := make(map[string]domainmodel.AgentConfigFile)
	for _, name := range asrc.RemoteConfig {
		configMap[name] = domainmodel.AgentConfigFile{
			Body:        nil,
			ContentType: "",
		}
	}

	return domainmodel.AgentSpecRemoteConfig{
		ConfigMap: domainmodel.AgentConfigMap{
			ConfigMap: configMap,
		},
	}
}

// ToDomainPtr converts the AgentSpecRemoteConfig to domain model pointer.
func (asrc *AgentSpecRemoteConfig) ToDomainPtr() *domainmodel.AgentSpecRemoteConfig {
	if asrc == nil || len(asrc.RemoteConfig) == 0 {
		return nil
	}

	// Convert RemoteConfig names to ConfigMap with empty placeholders
	configMap := make(map[string]domainmodel.AgentConfigFile)
	for _, name := range asrc.RemoteConfig {
		configMap[name] = domainmodel.AgentConfigFile{
			Body:        nil,
			ContentType: "",
		}
	}

	return &domainmodel.AgentSpecRemoteConfig{
		ConfigMap: domainmodel.AgentConfigMap{
			ConfigMap: configMap,
		},
	}
}

// ToDomain converts the AgentCustomCapabilities to domain model.
func (acc *AgentCustomCapabilities) ToDomain() *domainmodel.AgentCustomCapabilities {
	if acc == nil {
		return nil
	}

	return &domainmodel.AgentCustomCapabilities{
		Capabilities: acc.Capabilities,
	}
}

// ToDomain converts the AgentAvailableComponents to domain model.
func (avv *AgentAvailableComponents) ToDomain() *domainmodel.AgentAvailableComponents {
	if avv == nil {
		return nil
	}

	return &domainmodel.AgentAvailableComponents{
		Components: lo.MapValues(avv.Components,
			func(component ComponentDetails, _ string) domainmodel.ComponentDetails {
				return *component.ToDomain()
			}),
		Hash: avv.Hash.Data,
	}
}

// ToDomain converts the ComponentDetails to domain model.
func (cd *ComponentDetails) ToDomain() *domainmodel.ComponentDetails {
	if cd == nil {
		return nil
	}

	return &domainmodel.ComponentDetails{
		Metadata: cd.Metadata,
		SubComponentMap: lo.MapValues(cd.SubComponentMap,
			func(subComp ComponentDetails, _ string) domainmodel.ComponentDetails {
				return *subComp.ToDomain()
			}),
	}
}

// AgentFromDomain converts domain model to persistence model.
func AgentFromDomain(agent *domainmodel.Agent) *Agent {
	var newInstanceUID *bson.Binary
	if agent.Spec.NewInstanceUID != uuid.Nil {
		newInstanceUID = &bson.Binary{
			Subtype: bson.TypeBinaryUUID,
			Data:    agent.Spec.NewInstanceUID[:],
		}
	}

	return &Agent{
		Common: Common{
			Version: VersionV1,
			ID:      nil, // ID will be set by MongoDB
		},
		Metadata: AgentMetadata{
			InstanceUID: bson.Binary{
				Subtype: bson.TypeBinaryUUID,
				Data:    agent.Metadata.InstanceUID[:],
			},
			InstanceUIDString:  agent.Metadata.InstanceUID.String(),
			Capabilities:       AgentCapabilitiesFromDomain(&agent.Metadata.Capabilities),
			Description:        AgentDescriptionFromDomain(&agent.Metadata.Description),
			CustomCapabilities: AgentCustomCapabilitiesFromDomain(&agent.Metadata.CustomCapabilities),
		},
		Spec: AgentSpec{
			NewInstanceUID:      newInstanceUID,
			RemoteConfig:        AgentSpecRemoteConfigFromDomain(agent.Spec.RemoteConfig),
			RequiredRestartedAt: agentRestartInfoToBsonDateTime(agent.Spec.RestartInfo),
		},
		Status: AgentStatus{
			EffectiveConfig:     AgentEffectiveConfigFromDomain(&agent.Status.EffectiveConfig),
			PackageStatuses:     AgentPackageStatusesFromDomain(&agent.Status.PackageStatuses),
			ComponentHealth:     AgentComponentHealthFromDomain(&agent.Status.ComponentHealth),
			AvailableComponents: AgentAvailableComponentsFromDomain(&agent.Status.AvailableComponents),
			RemoteConfigStatus:  AgentRemoteConfigStatusFromDomain(&agent.Status.RemoteConfigStatus),
			Conditions:          AgentConditionsFromDomain(agent.Status.Conditions),
			Connected:           agent.Status.Connected,
			ConnectionType:      agent.Status.ConnectionType.String(),
			SequenceNum:         agent.Status.SequenceNum,
			LastCommunicatedAt:  bson.NewDateTimeFromTime(agent.Status.LastReportedAt),
			LastCommunicatedTo:  agent.Status.LastReportedTo,
		},
	}
}

func agentRestartInfoToBsonDateTime(restartInfo *domainmodel.AgentRestartInfo) bson.DateTime {
	if restartInfo == nil {
		return bson.NewDateTimeFromTime(time.Time{})
	}

	return bson.NewDateTimeFromTime(restartInfo.RequiredRestartedAt)
}

// AgentCapabilitiesFromDomain converts domain model to persistence model.
func AgentCapabilitiesFromDomain(ac *agent.Capabilities) *AgentCapabilities {
	if ac == nil {
		return nil
	}

	return (*AgentCapabilities)(ac)
}

// AgentDescriptionFromDomain converts domain model to persistence model.
func AgentDescriptionFromDomain(ads *agent.Description) *AgentDescription {
	if ads == nil {
		return nil
	}

	return &AgentDescription{
		IdentifyingAttributes:    MapToKeyValuePairs(ads.IdentifyingAttributes),
		NonIdentifyingAttributes: MapToKeyValuePairs(ads.NonIdentifyingAttributes),
	}
}

// AgentEffectiveConfigFromDomain converts domain model to persistence model.
func AgentEffectiveConfigFromDomain(aec *domainmodel.AgentEffectiveConfig) *AgentEffectiveConfig {
	if aec == nil {
		return nil
	}

	return &AgentEffectiveConfig{
		ConfigMap: AgentConfigMap{
			ConfigMap: lo.MapValues(aec.ConfigMap.ConfigMap,
				func(configFile domainmodel.AgentConfigFile, _ string) AgentConfigFile {
					return AgentConfigFile{
						Body: bson.Binary{
							Subtype: bson.TypeBinaryGeneric,
							Data:    configFile.Body,
						},
						ContentType: configFile.ContentType,
					}
				}),
		},
	}
}

// AgentPackageStatusesFromDomain converts domain model to persistence model.
func AgentPackageStatusesFromDomain(aps *domainmodel.AgentPackageStatuses) *AgentPackageStatuses {
	if aps == nil {
		return nil
	}

	return &AgentPackageStatuses{
		Packages: lo.MapValues(aps.Packages,
			func(pss domainmodel.AgentPackageStatusEntry, _ string) AgentPackageStatus {
				return AgentPackageStatus{
					Name:            pss.Name,
					AgentHasVersion: pss.AgentHasVersion,
					AgentHasHash: bson.Binary{
						Subtype: bson.TypeBinaryGeneric,
						Data:    pss.AgentHasHash,
					},
					ServerOfferedVersion: pss.ServerOfferedVersion,
					Status:               AgentPackageStatusEnum(pss.Status),
					ErrorMessage:         pss.ErrorMessage,
				}
			}),
		ServerProvidedAllPackgesHash: bson.Binary{
			Subtype: bson.TypeBinaryGeneric,
			Data:    aps.ServerProvidedAllPackgesHash,
		},
		ErrorMessage: aps.ErrorMessage,
	}
}

// AgentComponentHealthFromDomain converts domain model to persistence model.
func AgentComponentHealthFromDomain(ach *domainmodel.AgentComponentHealth) *AgentComponentHealth {
	if ach == nil {
		return nil
	}

	return &AgentComponentHealth{
		Healthy:             ach.Healthy,
		StartTimeUnixMilli:  ach.StartTime.UnixMilli(),
		LastError:           ach.LastError,
		Status:              ach.Status,
		StatusTimeUnixMilli: ach.StatusTime.UnixMilli(),
		ComponentHealthMap: lo.MapValues(ach.ComponentHealthMap,
			func(ach domainmodel.AgentComponentHealth, _ string) AgentComponentHealth {
				return *AgentComponentHealthFromDomain(&ach)
			}),
	}
}

// AgentSpecRemoteConfigFromDomain converts domain model to persistence model.
func AgentSpecRemoteConfigFromDomain(arc *domainmodel.AgentSpecRemoteConfig) *AgentSpecRemoteConfig {
	if arc == nil || len(arc.ConfigMap.ConfigMap) == 0 {
		return nil
	}

	// Extract config names from ConfigMap
	names := make([]string, 0, len(arc.ConfigMap.ConfigMap))
	for name := range arc.ConfigMap.ConfigMap {
		names = append(names, name)
	}

	return &AgentSpecRemoteConfig{
		RemoteConfig: names,
	}
}

// AgentCustomCapabilitiesFromDomain converts domain model to persistence model.
func AgentCustomCapabilitiesFromDomain(acc *domainmodel.AgentCustomCapabilities) *AgentCustomCapabilities {
	if acc == nil {
		return nil
	}

	return &AgentCustomCapabilities{
		Capabilities: acc.Capabilities,
	}
}

// ComponentDetailsFromDomain converts domain model to persistence model.
func ComponentDetailsFromDomain(cd *domainmodel.ComponentDetails) *ComponentDetails {
	return &ComponentDetails{
		Metadata: cd.Metadata,
		SubComponentMap: lo.MapValues(cd.SubComponentMap,
			func(subComp domainmodel.ComponentDetails, _ string) ComponentDetails {
				return *ComponentDetailsFromDomain(&subComp)
			}),
	}
}

// AgentAvailableComponentsFromDomain converts domain model to persistence model.
func AgentAvailableComponentsFromDomain(acc *domainmodel.AgentAvailableComponents) *AgentAvailableComponents {
	if acc == nil {
		return nil
	}

	return &AgentAvailableComponents{
		Components: lo.MapValues(acc.Components,
			func(cd domainmodel.ComponentDetails, _ string) ComponentDetails {
				return *ComponentDetailsFromDomain(&cd)
			}),
		Hash: bson.Binary{
			Subtype: bson.TypeBinaryGeneric,
			Data:    acc.Hash,
		},
	}
}

// ToDomain converts the AgentRemoteConfigStatus to domain model.
func (arcs *AgentRemoteConfigStatus) ToDomain() domainmodel.AgentRemoteConfigStatus {
	if arcs == nil {
		return domainmodel.AgentRemoteConfigStatus{
			LastRemoteConfigHash: nil,
			Status:               domainmodel.RemoteConfigStatusUnset,
			ErrorMessage:         "",
			LastUpdatedAt:        time.Time{},
		}
	}

	var lastRemoteConfigHash []byte
	if arcs.LastRemoteConfigHash != nil {
		lastRemoteConfigHash = arcs.LastRemoteConfigHash.Data
	}

	return domainmodel.AgentRemoteConfigStatus{
		LastRemoteConfigHash: lastRemoteConfigHash,
		Status:               domainmodel.RemoteConfigStatus(arcs.Status),
		ErrorMessage:         arcs.ErrorMessage,
		LastUpdatedAt:        time.Time{},
	}
}

// AgentRemoteConfigStatusFromDomain converts domain model to persistence model.
func AgentRemoteConfigStatusFromDomain(arcs *domainmodel.AgentRemoteConfigStatus) *AgentRemoteConfigStatus {
	if arcs == nil {
		return nil
	}

	var lastRemoteConfigHash *bson.Binary
	if arcs.LastRemoteConfigHash != nil {
		lastRemoteConfigHash = &bson.Binary{
			Subtype: bson.TypeBinaryGeneric,
			Data:    arcs.LastRemoteConfigHash,
		}
	}

	return &AgentRemoteConfigStatus{
		LastRemoteConfigHash: lastRemoteConfigHash,
		Status:               AgentRemoteConfigStatusEnum(arcs.Status),
		ErrorMessage:         arcs.ErrorMessage,
	}
}

// AgentConditionsFromDomain converts domain conditions to persistence model.
func AgentConditionsFromDomain(conditions []domainmodel.AgentCondition) []AgentCondition {
	if len(conditions) == 0 {
		return nil
	}

	result := make([]AgentCondition, len(conditions))
	for i, condition := range conditions {
		result[i] = AgentCondition{
			Type:               string(condition.Type),
			LastTransitionTime: bson.NewDateTimeFromTime(condition.LastTransitionTime),
			Status:             string(condition.Status),
			Reason:             condition.Reason,
			Message:            condition.Message,
		}
	}

	return result
}
