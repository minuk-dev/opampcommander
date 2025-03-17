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
	VersionV1 = 1
)

type Agent struct {
	// Magic Number of version
	Version int `json:"version"`

	InstanceUID         uuid.UUID                 `json:"instanceUid"`
	Capabilities        *AgentCapabilities        `json:"capabilities"`
	Description         *AgentDescription         `json:"description"`
	EffectiveConfig     *AgentEffectiveConfig     `json:"effectiveConfig"`
	PackageStatuses     *AgentPackageStatuses     `json:"packageStatuses"`
	ComponentHealth     *AgentComponentHealth     `json:"componentHealth"`
	RemoteConfig        *AgentRemoteConfig        `json:"remoteConfig"`
	CustomCapabilities  *AgentCustomCapabilities  `json:"customCapabilities"`
	AvailableComponents *AgentAvailableComponents `json:"availableComponents"`
}

type AgentDescription struct {
	IdentifyingAttributes    map[string]string `json:"identifyingAttributes"`
	NonIdentifyingAttributes map[string]string `json:"nonIdentifyingAttributes"`
}

type AgentComponentHealth struct {
	Healthy             bool                            `json:"healthy"`
	StartTimeUnixMilli  int64                           `json:"startTimeUnixMilli"`
	LastError           string                          `json:"lastError"`
	Status              string                          `json:"status"`
	StatusTimeUnixMilli int64                           `json:"statusTimeUnixMilli"`
	ComponentHealthMap  map[string]AgentComponentHealth `json:"componentHealthMap"`
}

type AgentCapabilities uint64

type AgentEffectiveConfig struct {
	ConfigMap AgentConfigMap `json:"configMap"`
}

type AgentConfigMap struct {
	ConfigMap map[string]AgentConfigFile `json:"configMap"`
}

type AgentConfigFile struct {
	Body        []byte `json:"body"`
	ContentType string `json:"contentType"`
}

type AgentRemoteConfig struct {
	RemoteConfigStatuses    []AgentRemoteConfigSub `json:"remoteConfigStatuses"`
	LastErrorMessage        string                 `json:"lastErrorMessage"`
	LastModifiedAtUnixMilli int64                  `json:"lastModifiedAtUnixMilli"`
}

type AgentRemoteConfigSub struct {
	Key   []byte                      `json:"key"`
	Value AgentRemoteConfigStatusEnum `json:"value"`
}

type AgentRemoteConfigStatusEnum int32

type AgentPackageStatuses struct {
	Packages                     map[string]AgentPackageStatus `json:"packages"`
	ServerProvidedAllPackgesHash []byte                        `json:"serverProvidedAllPackgesHash"`
	ErrorMessage                 string                        `json:"errorMessage"`
}

type AgentPackageStatus struct {
	Name                 string                 `json:"name"`
	AgentHasVersion      string                 `json:"agentHasVersion"`
	AgentHasHash         []byte                 `json:"agentHasHash"`
	ServerOfferedVersion string                 `json:"serverOfferedVersion"`
	Status               AgentPackageStatusEnum `json:"status"`
	ErrorMessage         string                 `json:"errorMessage"`
}

type AgentPackageStatusEnum int32

type AgentCustomCapabilities struct {
	Capabilities []string `json:"capabilities"`
}

type AgentAvailableComponents struct {
	Components map[string]ComponentDetails `json:"components"`
	Hash       []byte                      `json:"hash"`
}

type ComponentDetails struct {
	Metadata        map[string]string           `json:"metadata"`
	SubComponentMap map[string]ComponentDetails `json:"subComponentMap"`
}

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
	}
}

func (ac *AgentCapabilities) ToDomain() *domainmodel.AgentCapabilities {
	return (*domainmodel.AgentCapabilities)(ac)
}

func (ad *AgentDescription) ToDomain() *agent.Description {
	return &agent.Description{
		IdentifyingAttributes:    ad.IdentifyingAttributes,
		NonIdentifyingAttributes: ad.NonIdentifyingAttributes,
	}
}

func (ae *AgentEffectiveConfig) ToDomain() *domainmodel.AgentEffectiveConfig {
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

func (ap *AgentPackageStatuses) ToDomain() *domainmodel.AgentPackageStatuses {
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

func (ach *AgentComponentHealth) ToDomain() *domainmodel.AgentComponentHealth {
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

func (arc *AgentRemoteConfig) ToDomain() remoteconfig.RemoteConfig {
	remoteConfig := remoteconfig.New()
	if arc == nil {
		return remoteConfig
	}
	for _, sub := range arc.RemoteConfigStatuses {
		remoteConfig.SetStatus(remoteconfig.StatusWithKey{
			Key:   sub.Key,
			Value: remoteconfig.Status(sub.Value),
		})
	}

	remoteConfig.SetLastErrorMessage(arc.LastErrorMessage)
	remoteConfig.LastModifiedAt = time.UnixMilli(arc.LastModifiedAtUnixMilli)

	return remoteConfig
}

func (acc *AgentCustomCapabilities) ToDomain() *domainmodel.AgentCustomCapabilities {
	return &domainmodel.AgentCustomCapabilities{
		Capabilities: acc.Capabilities,
	}
}

func (avv *AgentAvailableComponents) ToDomain() *domainmodel.AgentAvailableComponents {
	return &domainmodel.AgentAvailableComponents{
		Components: lo.MapValues(avv.Components,
			func(component ComponentDetails, _ string) domainmodel.ComponentDetails {
				return *component.ToDomain()
			}),
		Hash: avv.Hash,
	}
}

func (cd *ComponentDetails) ToDomain() *domainmodel.ComponentDetails {
	return &domainmodel.ComponentDetails{
		Metadata: cd.Metadata,
		SubComponentMap: lo.MapValues(cd.SubComponentMap,
			func(subComp ComponentDetails, _ string) domainmodel.ComponentDetails {
				return *subComp.ToDomain()
			}),
	}
}

func AgentFromDomain(agent *domainmodel.Agent) *Agent {
	return &Agent{
		Version:             VersionV1,
		InstanceUID:         agent.InstanceUID,
		Capabilities:        AgentCapabilitiesFromDomain(agent.Capabilities),
		Description:         AgentDescriptionFromDomain(agent.Description),
		EffectiveConfig:     AgentEffectiveConfigFromDomain(agent.EffectiveConfig),
		PackageStatuses:     AgentPackageStatusesFromDomain(agent.PackageStatuses),
		ComponentHealth:     AgentComponentHealthFromDomain(agent.ComponentHealth),
		RemoteConfig:        AgentRemoteConfigFromDomain(agent.RemoteConfig),
		CustomCapabilities:  AgentCustomCapabilitiesFromDomain(agent.CustomCapabilities),
		AvailableComponents: AgentAvailableComponentsFromDomain(agent.AvailableComponents),
	}
}

func AgentCapabilitiesFromDomain(ac *domainmodel.AgentCapabilities) *AgentCapabilities {
	return (*AgentCapabilities)(ac)
}

func AgentDescriptionFromDomain(ad *agent.Description) *AgentDescription {
	return &AgentDescription{
		IdentifyingAttributes:    ad.IdentifyingAttributes,
		NonIdentifyingAttributes: ad.NonIdentifyingAttributes,
	}
}

func AgentEffectiveConfigFromDomain(aec *domainmodel.AgentEffectiveConfig) *AgentEffectiveConfig {
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

func AgentPackageStatusesFromDomain(aps *domainmodel.AgentPackageStatuses) *AgentPackageStatuses {
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

func AgentComponentHealthFromDomain(ach *domainmodel.AgentComponentHealth) *AgentComponentHealth {
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

func AgentRemoteConfigFromDomain(arc remoteconfig.RemoteConfig) *AgentRemoteConfig {
	statuses := arc.ListStatuses()
	if len(statuses) == 0 {
		return nil
	}

	remoteConfigStatuses := make([]AgentRemoteConfigSub, 0, len(statuses))
	for _, status := range statuses {
		remoteConfigStatuses = append(remoteConfigStatuses, AgentRemoteConfigSub{
			Key:   status.Key,
			Value: AgentRemoteConfigStatusEnum(status.Value),
		})
	}

	return &AgentRemoteConfig{
		RemoteConfigStatuses:    remoteConfigStatuses,
		LastErrorMessage:        arc.LastErrorMessage,
		LastModifiedAtUnixMilli: arc.LastModifiedAt.UnixMilli(),
	}
}

func AgentCustomCapabilitiesFromDomain(acc *domainmodel.AgentCustomCapabilities) *AgentCustomCapabilities {
	return &AgentCustomCapabilities{
		Capabilities: acc.Capabilities,
	}
}

func ComponentDetailsFromDomain(cd *domainmodel.ComponentDetails) *ComponentDetails {
	return &ComponentDetails{
		Metadata: cd.Metadata,
		SubComponentMap: lo.MapValues(cd.SubComponentMap,
			func(subComp domainmodel.ComponentDetails, _ string) ComponentDetails {
				return *ComponentDetailsFromDomain(&subComp)
			}),
	}
}

func AgentAvailableComponentsFromDomain(acc *domainmodel.AgentAvailableComponents) *AgentAvailableComponents {
	return &AgentAvailableComponents{
		Components: lo.MapValues(acc.Components,
			func(cd domainmodel.ComponentDetails, _ string) ComponentDetails {
				return *ComponentDetailsFromDomain(&cd)
			}),
		Hash: acc.Hash,
	}
}
