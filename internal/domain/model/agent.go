package model

import (
	"time"
)

// Agent is a domain model to control opamp agent by opampcommander.
type Agent struct {
	InstanceUUID        string
	Capabilities        *AgentCapabilities
	Description         *AgentDescription
	EffectiveConfig     *AgentEffectiveConfig
	PacakgeStatuses     *AgentPackageStatuses
	ComponentHealth     *AgentComponentHealth
	RemoteConfigStatus  *AgentRemoteConfigStatus
	CustomCapabilities  *AgentCustomCapabilities
	AvailableComponents *AgentAvailableComponents
}

type AgentDescription struct {
	IdentifyingAttributes    map[string]string
	NonIdentifyingAttributes map[string]string
}

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

type AgentCapabilities uint64

type AgentEffectiveConfig struct {
	ConfigMap AgentConfigMap
}

type AgentConfigMap struct {
	// The config_map field of the AgentConfigSet message is a map of configuration files, where keys are file names.
	ConfigMap map[string]AgentConfigFile
}

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

type AgentRemoteConfigStatus struct {
	LastRemoteConfigHash []byte
	Status               AgentRemoteConfigStatusEnum
	ErrorMessage         string
}

type AgentRemoteConfigStatusEnum int32

const (
	AgentRemoteConfigStatusEnumUnset    = 0
	AgentRemoteConfigStatusEnumApplied  = 1
	AgentRemoteConfigStatusEnumApplying = 2
	AgentRemoteConfigStatusEnumFailed   = 3
)

type AgentPackageStatuses struct {
	Packages                     map[string]AgentPackageStatus
	ServerProvidedAllPackgesHash []byte
	ErrorMessage                 string
}

type AgentPackageStatus struct {
	Name                 string
	AgentHasVersion      string
	AgentHasHash         []byte
	ServerOfferedVersion string
	Status               AgentPackageStatusEnum
	ErrorMessage         string
}

type AgentPackageStatusEnum int32

const (
	AgentPackageStatusEnumInstalled      = 0
	AgentPackageStatusEnumInstallPending = 1
	AgentPackageStatusEnumInstalling     = 2
	AgentPackageStatusEnumInstallFailed  = 3
	AgentPackageStatusEnumDownloading    = 4
)

type AgentCustomCapabilities struct {
	Capabilities []string
}

type AgentAvailableComponents struct {
	Components map[string]ComponentDetails
	Hash       []byte
}

type ComponentDetails struct {
	Metadata        map[string]string
	SubComponentMap map[string]ComponentDetails
}

type OS struct {
	Type    string
	Version string
}

type Service struct {
	Name       string
	Namespace  string
	Version    string
	InstanceID string
}

type AgentHost struct {
	Name string
}

func (a *Agent) ReportDescription(desc *AgentDescription) error {
	a.Description = desc

	return nil
}

func (a *Agent) ReportComponentHealth(health *AgentComponentHealth) error {
	a.ComponentHealth = health

	return nil
}

func (a *Agent) ReportEffectiveConfig(config *AgentEffectiveConfig) error {
	a.EffectiveConfig = config

	return nil
}

func (a *Agent) ReportRemoteConfigStatus(status *AgentRemoteConfigStatus) error {
	a.RemoteConfigStatus = status

	return nil
}

func (a *Agent) ReportPackageStatuses(status *AgentPackageStatuses) error {
	a.PacakgeStatuses = status

	return nil
}

func (a *Agent) ReportCustomCapabilities(capabilities *AgentCustomCapabilities) error {
	a.CustomCapabilities = capabilities

	return nil
}

func (a *Agent) ReportAvailableComponents(availableComponents *AgentAvailableComponents) error {
	a.AvailableComponents = availableComponents

	return nil
}

// OS is a required field of AgentDescription
// https://github.com/open-telemetry/opamp-spec/blob/main/specification.md#agentdescriptionnon_identifying_attributes
func (ad *AgentDescription) OS() OS {
	return OS{
		Type:    ad.NonIdentifyingAttributes["os.type"],
		Version: ad.NonIdentifyingAttributes["os.version"],
	}
}

func (ad *AgentDescription) Service() Service {
	return Service{
		Name:       ad.IdentifyingAttributes["service.name"],
		Namespace:  ad.IdentifyingAttributes["service.namespace"],
		Version:    ad.IdentifyingAttributes["service.version"],
		InstanceID: ad.IdentifyingAttributes["service.instance.id"],
	}
}

func (ad *AgentDescription) Host() AgentHost {
	return AgentHost{
		Name: ad.NonIdentifyingAttributes["host.name"],
	}
}
