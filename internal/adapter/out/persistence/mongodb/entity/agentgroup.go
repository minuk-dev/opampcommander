package entity

import (
	"time"

	"github.com/samber/lo"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

const (
	// AgentGroupKeyFieldName is the field name used as the key for AgentGroup entities in MongoDB.
	AgentGroupKeyFieldName string = "metadata.name"
)

// AgentGroup is the mongo entity representation of the AgentGroup domain model.
type AgentGroup struct {
	Common `bson:",inline"`

	Metadata AgentGroupMetadata `bson:"metadata"`
	Spec     AgentGroupSpec     `bson:"spec"`
	Status   AgentGroupStatus   `bson:"status"`
}

// AgentGroupMetadata represents metadata information for an agent group.
type AgentGroupMetadata struct {
	Name       string            `bson:"name"`
	Attributes map[string]string `bson:"attributes"`
	CreatedAt  *time.Time        `bson:"createdAt,omitempty"`
	DeletedAt  *time.Time        `bson:"deletedAt,omitempty"`
}

// AgentGroupSpec represents the specification of an agent group.
type AgentGroupSpec struct {
	Priority              int                          `bson:"priority"`
	Selector              AgentSelector                `bson:"selector"`
	AgentRemoteConfig     *AgentGroupAgentRemoteConfig `bson:"agentConfig,omitempty"`
	AgentConnectionConfig *AgentConnectionConfig       `bson:"agentConnectionConfig,omitempty"`
}

// AgentGroupStatus represents the status of an agent group in MongoDB.
type AgentGroupStatus struct {
	Conditions []Condition `bson:"conditions"`
}

// AgentSelector defines the criteria for selecting agents to be included in the agent group.
type AgentSelector struct {
	IdentifyingAttributes    map[string]string `json:"identifyingAttributes"`
	NonIdentifyingAttributes map[string]string `json:"nonIdentifyingAttributes"`
}

// AgentGroupAgentRemoteConfig represents the remote configuration for agents in the group.
type AgentGroupAgentRemoteConfig struct {
	AgentRemoteConfigName *string                `bson:"agentRemoteConfigName,omitempty"`
	AgentRemoteConfigSpec *AgentRemoteConfigSpec `bson:"agentRemoteConfigSpec,omitempty"`
	AgentRemoteConfigRef  *string                `bson:"agentRemoteConfigRef,omitempty"`
}

// AgentRemoteConfigSpec represents the specification of a remote config.
type AgentRemoteConfigSpec struct {
	Value       []byte `bson:"value"       json:"value"`
	ContentType string `bson:"contentType" json:"contentType"`
}

// AgentConnectionConfig represents connection settings for agents in the group.
type AgentConnectionConfig struct {
	OpAMP            ConnectionSettings            `bson:"opamp"            json:"opamp"`
	OwnMetrics       ConnectionSettings            `bson:"ownMetrics"       json:"ownMetrics"`
	OwnLogs          ConnectionSettings            `bson:"ownLogs"          json:"ownLogs"`
	OwnTraces        ConnectionSettings            `bson:"ownTraces"        json:"ownTraces"`
	OtherConnections map[string]ConnectionSettings `bson:"otherConnections" json:"otherConnections"`
}

// ConnectionSettings represents connection settings for telemetry.
type ConnectionSettings struct {
	DestinationEndpoint string              `bson:"destinationEndpoint"       json:"destinationEndpoint"`
	Headers             map[string][]string `bson:"headers,omitempty"         json:"headers,omitempty"`
	CertificateName     *string             `bson:"certificateName,omitempty" json:"certificateName,omitempty"`
}

// AgentGroupStatistics holds statistical data for an agent group.
type AgentGroupStatistics struct {
	NumAgents             int64
	NumConnectedAgents    int64
	NumHealthyAgents      int64
	NumUnhealthyAgents    int64
	NumNotConnectedAgents int64
}

// ToDomain converts the AgentGroup entity to the domain model.
func (e *AgentGroup) ToDomain(statistics *AgentGroupStatistics) *model.AgentGroup {
	agentGroup := &model.AgentGroup{
		Metadata: e.Metadata.toDomain(),
		Spec:     e.Spec.toDomain(),
		Status:   e.Status.toDomain(),
	}

	if statistics != nil {
		agentGroup.Status.NumAgents = int(statistics.NumAgents)
		agentGroup.Status.NumConnectedAgents = int(statistics.NumConnectedAgents)
		agentGroup.Status.NumHealthyAgents = int(statistics.NumHealthyAgents)
		agentGroup.Status.NumUnhealthyAgents = int(statistics.NumUnhealthyAgents)
		agentGroup.Status.NumNotConnectedAgents = int(statistics.NumNotConnectedAgents)
	}

	return agentGroup
}
func (s *AgentGroupMetadata) toDomain() model.AgentGroupMetadata {
	var deletedAt time.Time
	if s.DeletedAt != nil {
		deletedAt = *s.DeletedAt
	}

	var createdAt time.Time
	if s.CreatedAt != nil {
		createdAt = *s.CreatedAt
	}

	return model.AgentGroupMetadata{
		Name:       s.Name,
		Attributes: s.Attributes,
		CreatedAt:  createdAt,
		DeletedAt:  deletedAt,
	}
}

func (s *AgentGroupSpec) toDomain() model.AgentGroupSpec {
	//nolint:exhaustruct // Fields are set conditionally below
	spec := model.AgentGroupSpec{
		Priority: s.Priority,
		Selector: model.AgentSelector{
			IdentifyingAttributes:    s.Selector.IdentifyingAttributes,
			NonIdentifyingAttributes: s.Selector.NonIdentifyingAttributes,
		},
	}

	if s.AgentRemoteConfig != nil {
		spec.AgentRemoteConfig = &model.AgentGroupAgentRemoteConfig{
			AgentRemoteConfigName: s.AgentRemoteConfig.AgentRemoteConfigName,
			AgentRemoteConfigRef:  s.AgentRemoteConfig.AgentRemoteConfigRef,
			AgentRemoteConfigSpec: nil,
		}
		if s.AgentRemoteConfig.AgentRemoteConfigSpec != nil {
			spec.AgentRemoteConfig.AgentRemoteConfigSpec = &model.AgentRemoteConfigSpec{
				Value:       s.AgentRemoteConfig.AgentRemoteConfigSpec.Value,
				ContentType: s.AgentRemoteConfig.AgentRemoteConfigSpec.ContentType,
			}
		}
	}

	if s.AgentConnectionConfig != nil {
		spec.AgentConnectionConfig = &model.AgentGroupConnectionConfig{
			OpAMPConnection: &model.OpAMPConnectionSettings{
				DestinationEndpoint: s.AgentConnectionConfig.OpAMP.DestinationEndpoint,
				Headers:             s.AgentConnectionConfig.OpAMP.Headers,
				CertificateName:     s.AgentConnectionConfig.OpAMP.CertificateName,
			},
			OwnMetrics: &model.TelemetryConnectionSettings{
				DestinationEndpoint: s.AgentConnectionConfig.OwnMetrics.DestinationEndpoint,
				Headers:             s.AgentConnectionConfig.OwnMetrics.Headers,
				CertificateName:     s.AgentConnectionConfig.OwnMetrics.CertificateName,
			},
			OwnLogs: &model.TelemetryConnectionSettings{
				DestinationEndpoint: s.AgentConnectionConfig.OwnLogs.DestinationEndpoint,
				Headers:             s.AgentConnectionConfig.OwnLogs.Headers,
				CertificateName:     s.AgentConnectionConfig.OwnLogs.CertificateName,
			},
			OwnTraces: &model.TelemetryConnectionSettings{
				DestinationEndpoint: s.AgentConnectionConfig.OwnTraces.DestinationEndpoint,
				Headers:             s.AgentConnectionConfig.OwnTraces.Headers,
				CertificateName:     s.AgentConnectionConfig.OwnTraces.CertificateName,
			},
			OtherConnections: lo.MapValues(s.AgentConnectionConfig.OtherConnections,
				func(v ConnectionSettings, _ string) model.OtherConnectionSettings {
					return model.OtherConnectionSettings{
						DestinationEndpoint: v.DestinationEndpoint,
						Headers:             v.Headers,
						CertificateName:     v.CertificateName,
					}
				}),
		}
	}

	return spec
}

func (s *AgentGroupStatus) toDomain() model.AgentGroupStatus {
	//nolint:exhaustruct // Statistics fields are set by the caller
	return model.AgentGroupStatus{
		Conditions: lo.Map(s.Conditions, func(c Condition, _ int) model.Condition {
			return c.ToDomain()
		}),
	}
}

// AgentGroupFromDomain converts the AgentGroup domain model to the entity representation.
func AgentGroupFromDomain(agentgroup *model.AgentGroup) *AgentGroup {
	return &AgentGroup{
		Common: Common{
			Version: VersionV1,
			ID:      nil, // ID will be set by MongoDB
		},
		Metadata: agentGroupMetadataFromDomain(agentgroup.Metadata),
		Spec:     agentGroupSpecFromDomain(agentgroup.Spec),
		Status:   agentGroupStatusFromDomain(agentgroup.Status),
	}
}

func agentGroupMetadataFromDomain(metadata model.AgentGroupMetadata) AgentGroupMetadata {
	var deletedAt *time.Time
	if !metadata.DeletedAt.IsZero() {
		deletedAt = &metadata.DeletedAt
	}

	var createdAt *time.Time
	if !metadata.CreatedAt.IsZero() {
		createdAt = &metadata.CreatedAt
	}

	return AgentGroupMetadata{
		Name:       metadata.Name,
		Attributes: metadata.Attributes,
		CreatedAt:  createdAt,
		DeletedAt:  deletedAt,
	}
}

func agentGroupSpecFromDomain(spec model.AgentGroupSpec) AgentGroupSpec {
	//nolint:exhaustruct // Fields are set conditionally below
	result := AgentGroupSpec{
		Priority: spec.Priority,
		Selector: AgentSelector{
			IdentifyingAttributes:    spec.Selector.IdentifyingAttributes,
			NonIdentifyingAttributes: spec.Selector.NonIdentifyingAttributes,
		},
	}

	if spec.AgentRemoteConfig != nil {
		result.AgentRemoteConfig = &AgentGroupAgentRemoteConfig{
			AgentRemoteConfigName: spec.AgentRemoteConfig.AgentRemoteConfigName,
			AgentRemoteConfigRef:  spec.AgentRemoteConfig.AgentRemoteConfigRef,
			AgentRemoteConfigSpec: nil,
		}
		if spec.AgentRemoteConfig.AgentRemoteConfigSpec != nil {
			result.AgentRemoteConfig.AgentRemoteConfigSpec = &AgentRemoteConfigSpec{
				Value:       spec.AgentRemoteConfig.AgentRemoteConfigSpec.Value,
				ContentType: spec.AgentRemoteConfig.AgentRemoteConfigSpec.ContentType,
			}
		}
	}

	if spec.AgentConnectionConfig != nil {
		result.AgentConnectionConfig = &AgentConnectionConfig{
			OpAMP: ConnectionSettings{
				DestinationEndpoint: spec.AgentConnectionConfig.OpAMPConnection.DestinationEndpoint,
				Headers:             spec.AgentConnectionConfig.OpAMPConnection.Headers,
				CertificateName:     spec.AgentConnectionConfig.OpAMPConnection.CertificateName,
			},
			OwnMetrics: ConnectionSettings{
				DestinationEndpoint: spec.AgentConnectionConfig.OwnMetrics.DestinationEndpoint,
				Headers:             spec.AgentConnectionConfig.OwnMetrics.Headers,
				CertificateName:     spec.AgentConnectionConfig.OwnMetrics.CertificateName,
			},
			OwnLogs: ConnectionSettings{
				DestinationEndpoint: spec.AgentConnectionConfig.OwnLogs.DestinationEndpoint,
				Headers:             spec.AgentConnectionConfig.OwnLogs.Headers,
				CertificateName:     spec.AgentConnectionConfig.OwnLogs.CertificateName,
			},
			OwnTraces: ConnectionSettings{
				DestinationEndpoint: spec.AgentConnectionConfig.OwnTraces.DestinationEndpoint,
				Headers:             spec.AgentConnectionConfig.OwnTraces.Headers,
				CertificateName:     spec.AgentConnectionConfig.OwnTraces.CertificateName,
			},
			OtherConnections: lo.MapValues(spec.AgentConnectionConfig.OtherConnections,
				func(v model.OtherConnectionSettings, _ string) ConnectionSettings {
					return ConnectionSettings{
						DestinationEndpoint: v.DestinationEndpoint,
						Headers:             v.Headers,
						CertificateName:     v.CertificateName,
					}
				}),
		}
	}

	return result
}

func agentGroupStatusFromDomain(status model.AgentGroupStatus) AgentGroupStatus {
	return AgentGroupStatus{
		Conditions: lo.Map(status.Conditions, func(c model.Condition, _ int) Condition {
			return NewConditionFromDomain(c)
		}),
	}
}
