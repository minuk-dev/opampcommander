package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model/agent"
	"github.com/minuk-dev/opampcommander/internal/domain/model/remoteconfig"
	"github.com/minuk-dev/opampcommander/internal/domain/model/vo"
)

// Agent is a domain model to control opamp agent by opampcommander.
type Agent struct {
	InstanceUID         uuid.UUID
	Capabilities        *AgentCapabilities
	Description         *agent.Description
	EffectiveConfig     *AgentEffectiveConfig
	PackageStatuses     *AgentPackageStatuses
	ComponentHealth     *AgentComponentHealth
	RemoteConfig        remoteconfig.RemoteConfig
	CustomCapabilities  *AgentCustomCapabilities
	AvailableComponents *AgentAvailableComponents
}

// AgentComponentHealth is a domain model to control opamp agent component health.
type AgentComponentHealth struct {
	// Set to true if the Agent is up and healthy.
	Healthy bool

	// Timestamp since the Agent is up
	StartTime time.Time

	// Human-readable error message if the Agent is in erroneous state. SHOULD be set when healthy==false.
	LastError string

	// Component status represented as a string.
	// The status values are defined by agent-specific semantics and not at the protocol level.
	Status string

	// The time when the component status was observed.
	StatusTime time.Time

	// A map to store more granular, sub-component health.
	// It can nest as deeply as needed to describe the underlying system.
	ComponentHealthMap map[string]AgentComponentHealth
}

// AgentCapabilities is a bitmask of capabilities that the Agent supports.
// The AgentCapabilities enum is defined in the opamp protocol.
type AgentCapabilities uint64

// AgentEffectiveConfig is the effective configuration of the agent.
type AgentEffectiveConfig struct {
	ConfigMap AgentConfigMap
}

// AgentConfigMap is a map of configuration files.
type AgentConfigMap struct {
	// The config_map field of the AgentConfigSet message is a map of configuration files, where keys are file names.
	ConfigMap map[string]AgentConfigFile
}

// AgentConfigFile is a configuration file.
type AgentConfigFile struct {
	// The body field contains the raw bytes of the configuration file.
	// The content, format and encoding of the raw bytes is Agent type-specific and is outside the concerns of OpAMP
	// protocol.
	Body []byte

	// content_type is an optional field. It is a MIME Content-Type that describes what's contained in the body field,
	// for example "text/yaml". The content_type reported in the Effective Configuration in the Agent's status report may
	// be used for example by the Server to visualize the reported configuration nicely in a UI.
	ContentType string
}

// AgentRemoteConfigStatus is the status of the remote configuration.
type AgentRemoteConfigStatus struct {
	LastRemoteConfigHash []byte
	Status               remoteconfig.Status
	ErrorMessage         string
}

// AgentPackageStatuses is a map of package statuses.
type AgentPackageStatuses struct {
	Packages                     map[string]AgentPackageStatus
	ServerProvidedAllPackgesHash []byte
	ErrorMessage                 string
}

// AgentPackageStatus is the status of a package.
type AgentPackageStatus struct {
	Name                 string
	AgentHasVersion      string
	AgentHasHash         []byte
	ServerOfferedVersion string
	Status               AgentPackageStatusEnum
	ErrorMessage         string
}

// AgentPackageStatusEnum is an enum that represents the status of a package.
type AgentPackageStatusEnum int32

// AgentPackageStatusEnum values
// The AgentPackageStatusEnum enum is defined in the opamp protocol.
const (
	AgentPackageStatusEnumInstalled      = 0
	AgentPackageStatusEnumInstallPending = 1
	AgentPackageStatusEnumInstalling     = 2
	AgentPackageStatusEnumInstallFailed  = 3
	AgentPackageStatusEnumDownloading    = 4
)

// AgentCustomCapabilities is a list of custom capabilities that the Agent supports.
type AgentCustomCapabilities struct {
	Capabilities []string
}

// AgentAvailableComponents is a map of available components.
type AgentAvailableComponents struct {
	Components map[string]ComponentDetails
	Hash       []byte
}

// ComponentDetails is a details of a component.
type ComponentDetails struct {
	Metadata        map[string]string
	SubComponentMap map[string]ComponentDetails
}

// ReportDescription is a method to report the description of the agent.
func (a *Agent) ReportDescription(desc *agent.Description) error {
	if desc == nil {
		return nil // No description to report
	}

	a.Description = desc

	return nil
}

// ReportComponentHealth is a method to report the component health of the agent.
func (a *Agent) ReportComponentHealth(health *AgentComponentHealth) error {
	if health == nil {
		return nil // No health to report
	}

	a.ComponentHealth = health

	return nil
}

// ReportEffectiveConfig is a method to report the effective configuration of the agent.
func (a *Agent) ReportEffectiveConfig(config *AgentEffectiveConfig) error {
	if config == nil {
		return nil // No effective config to report
	}

	a.EffectiveConfig = config

	return nil
}

// ReportRemoteConfigStatus is a method to report the remote configuration status of the agent.
func (a *Agent) ReportRemoteConfigStatus(status *AgentRemoteConfigStatus) error {
	if status == nil {
		return nil // No remote config status to report
	}

	if status.ErrorMessage != "" {
		a.RemoteConfig.SetLastErrorMessage(status.ErrorMessage)
	}

	a.RemoteConfig.SetStatus(
		vo.Hash(status.LastRemoteConfigHash),
		status.Status,
	)

	return nil
}

// ApplyRemoteConfig is a method to apply the remote configuration to the agent.
func (a *Agent) ApplyRemoteConfig(config any) error {
	subconfig, err := remoteconfig.NewCommand(config)
	if err != nil {
		return fmt.Errorf("failed to create remote config command: %w", err)
	}

	err = a.RemoteConfig.ApplyRemoteConfig(subconfig)
	if err != nil {
		return fmt.Errorf("failed to apply remote config: %w", err)
	}

	return nil
}

// ReportPackageStatuses is a method to report the package statuses of the agent.
func (a *Agent) ReportPackageStatuses(status *AgentPackageStatuses) error {
	if status == nil {
		return nil // No package statuses to report
	}

	a.PackageStatuses = status

	return nil
}

// ReportCustomCapabilities is a method to report the custom capabilities of the agent.
func (a *Agent) ReportCustomCapabilities(capabilities *AgentCustomCapabilities) error {
	if capabilities == nil {
		return nil // No custom capabilities to report
	}

	a.CustomCapabilities = capabilities

	return nil
}

// ReportAvailableComponents is a method to report the available components of the agent.
func (a *Agent) ReportAvailableComponents(availableComponents *AgentAvailableComponents) error {
	if availableComponents == nil {
		return nil // No available components to report
	}

	a.AvailableComponents = availableComponents

	return nil
}
