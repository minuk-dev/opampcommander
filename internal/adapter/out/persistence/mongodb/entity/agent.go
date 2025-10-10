package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	domainmodel "github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/agent"
	"github.com/minuk-dev/opampcommander/internal/domain/model/remoteconfig"
)

const (
	// AgentKeyFieldName is the field name used as the key for Agent entities in MongoDB.
	AgentKeyFieldName string = "metadata.instanceUid"
)

// Agent is a struct that represents the MongoDB entity for an Agent.
type Agent struct {
	Common `bson:",inline"`

	Metadata AgentMetadata `bson:"metadata"`
	Spec     AgentSpec     `bson:"spec"`
	Status   AgentStatus   `bson:"status"`
	Commands AgentCommands `bson:"commands"`
}

// AgentMetadata represents the metadata of an agent.
type AgentMetadata struct {
	InstanceUID        uuid.UUID                `bson:"instanceUid"`
	Capabilities       *AgentCapabilities       `bson:"capabilities,omitempty"`
	Description        *AgentDescription        `bson:"description,omitempty"`
	CustomCapabilities *AgentCustomCapabilities `bson:"customCapabilities,omitempty"`
}

// AgentSpec represents the desired specification of an agent.
type AgentSpec struct {
	RemoteConfig *AgentRemoteConfig `bson:"remoteConfig,omitempty"`
}

// AgentStatus represents the current status of an agent.
type AgentStatus struct {
	EffectiveConfig     *AgentEffectiveConfig     `bson:"effectiveConfig,omitempty"`
	PackageStatuses     *AgentPackageStatuses     `bson:"packageStatuses,omitempty"`
	ComponentHealth     *AgentComponentHealth     `bson:"componentHealth,omitempty"`
	AvailableComponents *AgentAvailableComponents `bson:"availableComponents,omitempty"`
}

// AgentCommands represents the commands to be sent to an agent.
type AgentCommands struct {
	Commands []AgentCommand `bson:"commands"`
}

// AgentCommand represents a command to be sent to an agent.
type AgentCommand struct {
	CommandID       uuid.UUID     `bson:"commandId"`
	ReportFullState bool          `bson:"reportFullState"`
	CreatedAt       bson.DateTime `bson:"createdAt"`
	CreatedBy       string        `bson:"createdBy"`
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
	Body        []byte `bson:"body"`
	ContentType string `bson:"contentType"`
}

// AgentRemoteConfig is a struct to manage remote config.
type AgentRemoteConfig struct {
	RemoteConfigStatuses    []AgentRemoteConfigSub `bson:"remoteConfigStatuses"`
	LastErrorMessage        string                 `bson:"lastErrorMessage"`
	LastModifiedAtUnixMilli int64                  `bson:"lastModifiedAtUnixMilli"`
}

// AgentRemoteConfigSub is a struct to manage remote config status with key.
type AgentRemoteConfigSub struct {
	Key   []byte                      `bson:"key"`
	Value AgentRemoteConfigStatusEnum `bson:"value"`
}

// AgentRemoteConfigStatusEnum is an enum that represents the status of the remote config.
type AgentRemoteConfigStatusEnum int32

// AgentPackageStatuses is a map of package statuses.
type AgentPackageStatuses struct {
	Packages                     map[string]AgentPackageStatus `bson:"packages"`
	ServerProvidedAllPackgesHash []byte                        `bson:"serverProvidedAllPackgesHash"`
	ErrorMessage                 string                        `bson:"errorMessage"`
}

// AgentPackageStatus is a status of a package.
type AgentPackageStatus struct {
	Name                 string                 `bson:"name"`
	AgentHasVersion      string                 `bson:"agentHasVersion"`
	AgentHasHash         []byte                 `bson:"agentHasHash"`
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
	Hash       []byte                      `bson:"hash"`
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
		Commands: a.Commands.ToDomain(),
	}
}

// ToDmain converts the AgentMetadata to domain model.
func (metadata *AgentMetadata) ToDmain() domainmodel.AgentMetadata {
	return domainmodel.AgentMetadata{
		InstanceUID: metadata.InstanceUID,
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
	return domainmodel.AgentSpec{
		RemoteConfig: spec.RemoteConfig.ToDomain(),
	}
}

// ToDomain converts the AgentStatus to domain model.
func (status *AgentStatus) ToDomain() domainmodel.AgentStatus {
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
	}
}

// ToDomain converts the AgentCommands to domain model.
func (ac *AgentCommands) ToDomain() domainmodel.AgentCommands {
	if ac == nil {
		return domainmodel.AgentCommands{
			Commands: []domainmodel.AgentCommand{},
		}
	}

	commands := make([]domainmodel.AgentCommand, 0, len(ac.Commands))
	for _, cmd := range ac.Commands {
		commands = append(commands, domainmodel.AgentCommand{
			CommandID:       cmd.CommandID,
			ReportFullState: cmd.ReportFullState,
			CreatedAt:       cmd.CreatedAt.Time(),
			CreatedBy:       cmd.CreatedBy,
		})
	}

	return domainmodel.AgentCommands{
		Commands: commands,
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
					Body:        acf.Body,
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
		Packages: lo.MapValues(ap.Packages, func(aps AgentPackageStatus, _ string) domainmodel.AgentPackageStatus {
			return domainmodel.AgentPackageStatus{
				Name:                 aps.Name,
				AgentHasVersion:      aps.AgentHasVersion,
				AgentHasHash:         aps.AgentHasHash,
				ServerOfferedVersion: aps.ServerOfferedVersion,
				Status:               domainmodel.AgentPackageStatusEnum(aps.Status),
				ErrorMessage:         aps.ErrorMessage,
			}
		}),
		ServerProvidedAllPackgesHash: ap.ServerProvidedAllPackgesHash,
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

// ToDomain converts the AgentRemoteConfig to domain model.
func (arc *AgentRemoteConfig) ToDomain() remoteconfig.RemoteConfig {
	remoteConfig := remoteconfig.New()
	if arc == nil {
		return remoteConfig
	}

	for _, sub := range arc.RemoteConfigStatuses {
		remoteConfig.SetStatus(sub.Key, remoteconfig.Status(sub.Value))
	}

	remoteConfig.SetLastErrorMessage(arc.LastErrorMessage)
	remoteConfig.LastModifiedAt = time.UnixMilli(arc.LastModifiedAtUnixMilli)

	return remoteConfig
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
		Hash: avv.Hash,
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
	return &Agent{
		Common: Common{
			Version: VersionV1,
			ID:      nil, // ID will be set by MongoDB
		},
		Metadata: AgentMetadata{
			InstanceUID:        agent.Metadata.InstanceUID,
			Capabilities:       AgentCapabilitiesFromDomain(&agent.Metadata.Capabilities),
			Description:        AgentDescriptionFromDomain(&agent.Metadata.Description),
			CustomCapabilities: AgentCustomCapabilitiesFromDomain(&agent.Metadata.CustomCapabilities),
		},
		Spec: AgentSpec{
			RemoteConfig: AgentRemoteConfigFromDomain(agent.Spec.RemoteConfig),
		},
		Status: AgentStatus{
			EffectiveConfig:     AgentEffectiveConfigFromDomain(&agent.Status.EffectiveConfig),
			PackageStatuses:     AgentPackageStatusesFromDomain(&agent.Status.PackageStatuses),
			ComponentHealth:     AgentComponentHealthFromDomain(&agent.Status.ComponentHealth),
			AvailableComponents: AgentAvailableComponentsFromDomain(&agent.Status.AvailableComponents),
		},
		Commands: AgentCommandsFromDomain(&agent.Commands),
	}
}

// AgentCommandsFromDomain converts domain model to persistence model.
func AgentCommandsFromDomain(domain *domainmodel.AgentCommands) AgentCommands {
	if domain == nil {
		return AgentCommands{
			Commands: []AgentCommand{},
		}
	}

	commands := make([]AgentCommand, 0, len(domain.Commands))
	for _, cmd := range domain.Commands {
		commands = append(commands, AgentCommand{
			CommandID:       cmd.CommandID,
			ReportFullState: cmd.ReportFullState,
			CreatedAt:       bson.NewDateTimeFromTime(cmd.CreatedAt),
			CreatedBy:       cmd.CreatedBy,
		})
	}

	return AgentCommands{
		Commands: commands,
	}
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
				func(cf domainmodel.AgentConfigFile, _ string) AgentConfigFile {
					return AgentConfigFile{
						Body:        cf.Body,
						ContentType: cf.ContentType,
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
			func(pss domainmodel.AgentPackageStatus, _ string) AgentPackageStatus {
				return AgentPackageStatus{
					Name:                 pss.Name,
					AgentHasVersion:      pss.AgentHasVersion,
					AgentHasHash:         pss.AgentHasHash,
					ServerOfferedVersion: pss.ServerOfferedVersion,
					Status:               AgentPackageStatusEnum(pss.Status),
					ErrorMessage:         pss.ErrorMessage,
				}
			}),
		ServerProvidedAllPackgesHash: aps.ServerProvidedAllPackgesHash,
		ErrorMessage:                 aps.ErrorMessage,
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

// AgentRemoteConfigFromDomain converts domain model to persistence model.
func AgentRemoteConfigFromDomain(arc remoteconfig.RemoteConfig) *AgentRemoteConfig {
	statuses := arc.ListStatuses()
	if len(statuses) == 0 {
		return nil
	}

	remoteConfigStatuses := make([]AgentRemoteConfigSub, 0, len(statuses))
	for _, status := range statuses {
		remoteConfigStatuses = append(remoteConfigStatuses, AgentRemoteConfigSub{
			Key:   status.Key,
			Value: AgentRemoteConfigStatusEnum(status.Status),
		})
	}

	return &AgentRemoteConfig{
		RemoteConfigStatuses:    remoteConfigStatuses,
		LastErrorMessage:        arc.LastErrorMessage,
		LastModifiedAtUnixMilli: arc.LastModifiedAt.UnixMilli(),
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
		Hash: acc.Hash,
	}
}
