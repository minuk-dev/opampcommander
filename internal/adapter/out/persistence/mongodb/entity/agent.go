package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	domainmodel "github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/agent"
	"github.com/minuk-dev/opampcommander/internal/domain/model/remoteconfig"
)

const (
	// AgentKeyFieldName is the field name used as the key for Agent entities in MongoDB.
	AgentKeyFieldName string = "instanceUid"
)

// Agent is a struct that represents the MongoDB entity for an Agent.
type Agent struct {
	Common `bson:",inline"`

	InstanceUID         uuid.UUID                 `bson:"instanceUid"`
	Capabilities        *AgentCapabilities        `bson:"capabilities,omitempty"`
	Description         *AgentDescription         `bson:"description,omitempty"`
	EffectiveConfig     *AgentEffectiveConfig     `bson:"effectiveConfig,omitempty"`
	PackageStatuses     *AgentPackageStatuses     `bson:"packageStatuses,omitempty"`
	ComponentHealth     *AgentComponentHealth     `bson:"componentHealth,omitempty"`
	RemoteConfig        *AgentRemoteConfig        `bson:"remoteConfig,omitempty"`
	CustomCapabilities  *AgentCustomCapabilities  `bson:"customCapabilities,omitempty"`
	AvailableComponents *AgentAvailableComponents `bson:"availableComponents,omitempty"`

	ReportFullState bool `bson:"reportFullState"`
}

// AgentDescription is a struct to manage agent description.
type AgentDescription struct {
	IdentifyingAttributes    map[string]string `bson:"identifyingAttributes,omitempty"`
	NonIdentifyingAttributes map[string]string `bson:"nonIdentifyingAttributes,omitempty"`
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
		InstanceUID:         a.InstanceUID,
		Capabilities:        a.Capabilities.ToDomain(),
		Description:         a.Description.ToDomain(),
		EffectiveConfig:     a.EffectiveConfig.ToDomain(),
		PackageStatuses:     a.PackageStatuses.ToDomain(),
		ComponentHealth:     a.ComponentHealth.ToDomain(),
		RemoteConfig:        a.RemoteConfig.ToDomain(),
		CustomCapabilities:  a.CustomCapabilities.ToDomain(),
		AvailableComponents: a.AvailableComponents.ToDomain(),

		ReportFullState: a.ReportFullState,
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
		IdentifyingAttributes:    ad.IdentifyingAttributes,
		NonIdentifyingAttributes: ad.NonIdentifyingAttributes,
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
		InstanceUID:         agent.InstanceUID,
		Capabilities:        AgentCapabilitiesFromDomain(agent.Capabilities),
		Description:         AgentDescriptionFromDomain(agent.Description),
		EffectiveConfig:     AgentEffectiveConfigFromDomain(agent.EffectiveConfig),
		PackageStatuses:     AgentPackageStatusesFromDomain(agent.PackageStatuses),
		ComponentHealth:     AgentComponentHealthFromDomain(agent.ComponentHealth),
		RemoteConfig:        AgentRemoteConfigFromDomain(agent.RemoteConfig),
		CustomCapabilities:  AgentCustomCapabilitiesFromDomain(agent.CustomCapabilities),
		AvailableComponents: AgentAvailableComponentsFromDomain(agent.AvailableComponents),

		ReportFullState: agent.ReportFullState,
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
		IdentifyingAttributes:    ads.IdentifyingAttributes,
		NonIdentifyingAttributes: ads.NonIdentifyingAttributes,
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
