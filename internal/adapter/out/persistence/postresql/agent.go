package postresql

import (
	"context"

	"github.com/google/uuid"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
	"gorm.io/gorm"
)

var _ domainport.AgentPersistencePort = (*AgentPostgreAdapter)(nil)

type Agent struct {
	gorm.Model

	Version int

	InstanceUID         uuid.UUID
	Capabilities        AgentCapacities          `gorm:"embedded"`
	Description         AgentDescription         `gorm:"embedded"`
	EffectiveConfig     AgentEffectiveConfig     `gorm:"embedded"`
	PackageStatuses     AgentPackageStatuses     `gorm:"embedded"`
	ComponentHealth     AgentComponentHealth     `gorm:"embedded"`
	RemoteConfig        AgentRemoteConfig        `gorm:"embedded"`
	CustomCapabilities  AgentCustomCapabilities  `gorm:"embedded"`
	AvailableComponents AgentAvailableComponents `gorm:"embedded"`

	ReportFullState bool
}

// AgentComponentHealth is a struct to manage component health.
type AgentComponentHealth struct {
	Healthy             bool
	StartTimeUnixMilli  int64
	LastError           string
	Status              string
	StatusTimeUnixMilli int64
	ComponentHealthMap  map[string]AgentComponentHealth
}

// AgentEffectiveConfig is a struct to manage effective config.
type AgentEffectiveConfig struct {
	ConfigMap AgentConfigMap `json:"configMap"`
}

// AgentConfigMap is a struct to manage config map.
type AgentConfigMap struct {
	ConfigMap map[string]AgentConfigFile `json:"configMap"`
}

// AgentConfigFile is a struct to manage config file.
type AgentConfigFile struct {
	Body        []byte `json:"body"`
	ContentType string `json:"contentType"`
}

// AgentRemoteConfig is a struct to manage remote config.
type AgentRemoteConfig struct {
	RemoteConfigStatuses    []AgentRemoteConfigSub
	LastErrorMessage        string
	LastModifiedAtUnixMilli int64
}

// AgentRemoteConfigSub is a struct to manage remote config status with key.
type AgentRemoteConfigSub struct {
	Key   []byte                      `json:"key"`
	Value AgentRemoteConfigStatusEnum `json:"value"`
}

// AgentRemoteConfigStatusEnum is an enum that represents the status of the remote config.
type AgentRemoteConfigStatusEnum int32

// AgentPackageStatuses is a map of package statuses.
type AgentPackageStatuses struct {
	Packages                     map[string]AgentPackageStatus `json:"packages"`
	ServerProvidedAllPackgesHash []byte                        `json:"serverProvidedAllPackgesHash"`
	ErrorMessage                 string                        `json:"errorMessage"`
}

// AgentPackageStatus is a status of a package.
type AgentPackageStatus struct {
	Name                 string                 `json:"name"`
	AgentHasVersion      string                 `json:"agentHasVersion"`
	AgentHasHash         []byte                 `json:"agentHasHash"`
	ServerOfferedVersion string                 `json:"serverOfferedVersion"`
	Status               AgentPackageStatusEnum `json:"status"`
	ErrorMessage         string                 `json:"errorMessage"`
}

// AgentPackageStatusEnum is an enum that represents the status of a package.
type AgentPackageStatusEnum int32

// AgentCustomCapabilities is a custom capabilities of the agent.
type AgentCustomCapabilities struct {
	Capabilities []string `json:"capabilities"`
}

// AgentAvailableComponents is a map of available components.
type AgentAvailableComponents struct {
	Components map[string]ComponentDetails `json:"components"`
	Hash       []byte                      `json:"hash"`
}

// ComponentDetails is a details of a component.
type ComponentDetails struct {
	Metadata        map[string]string           `json:"metadata"`
	SubComponentMap map[string]ComponentDetails `json:"subComponentMap"`
}

type AgentPostgreAdapter struct {
}

// GetAgent implements port.AgentPersistencePort.
func (a *AgentPostgreAdapter) GetAgent(ctx context.Context, instanceUID uuid.UUID) (*model.Agent, error) {
	panic("unimplemented")
}

// ListAgents implements port.AgentPersistencePort.
func (a *AgentPostgreAdapter) ListAgents(ctx context.Context, options *model.ListOptions) (*model.ListResponse[*model.Agent], error) {
	panic("unimplemented")
}

// PutAgent implements port.AgentPersistencePort.
func (a *AgentPostgreAdapter) PutAgent(ctx context.Context, agent *model.Agent) error {
	panic("unimplemented")
}
