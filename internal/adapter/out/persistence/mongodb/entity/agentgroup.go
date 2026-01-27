package entity

import (
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

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
	Priority   int               `bson:"priority"`
	Attributes map[string]string `bson:"attributes"`
	Selector   AgentSelector     `bson:"selector"`
}

// AgentGroupSpec represents the specification of an agent group.
type AgentGroupSpec struct {
	AgentRemoteConfig     *AgentRemoteConfig     `bson:"agentConfig,omitempty"`
	AgentConnectionConfig *AgentConnectionConfig `bson:"agentConnectionConfig,omitempty"`
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

// AgentRemoteConfig represents the remote configuration for agents in the group.
type AgentRemoteConfig struct {
	Value       string `bson:"value"       json:"value"`
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
	DestinationEndpoint string                   `bson:"destinationEndpoint"   json:"destinationEndpoint"`
	Headers             map[string][]string      `bson:"headers,omitempty"     json:"headers,omitempty"`
	Certificate         TelemetryTLSCeritificate `bson:"certificate,omitempty" json:"certificate,omitempty"`
}

// TelemetryTLSCeritificate represents TLS certificate for telemetry connections.
type TelemetryTLSCeritificate struct {
	Cert       bson.Binary `bson:"cert,omitempty"       json:"cert,omitempty"`
	PrivateKey bson.Binary `bson:"privateKey,omitempty" json:"privateKey,omitempty"`
	CaCert     bson.Binary `bson:"caCert,omitempty"     json:"caCert,omitempty"`
}

// NewTelemetryTLSCertificate creates a new TelemetryTLSCeritificate from domain model.
func NewTelemetryTLSCertificate(domain model.TelemetryTLSCertificate) TelemetryTLSCeritificate {
	return TelemetryTLSCeritificate{
		//nolint:exhaustruct // Subtype is optional for generic binary data
		Cert: bson.Binary{Data: domain.Cert},
		//nolint:exhaustruct // Subtype is optional for generic binary data
		PrivateKey: bson.Binary{Data: domain.PrivateKey},
		//nolint:exhaustruct // Subtype is optional for generic binary data
		CaCert: bson.Binary{Data: domain.CaCert},
	}
}

// ToDomain converts TelemetryTLSCeritificate to domain model.
func (tc *TelemetryTLSCeritificate) ToDomain() model.TelemetryTLSCertificate {
	return model.TelemetryTLSCertificate{
		Cert:       tc.Cert.Data,
		PrivateKey: tc.PrivateKey.Data,
		CaCert:     tc.CaCert.Data,
	}
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
	return model.AgentGroupMetadata{
		Name:       s.Name,
		Priority:   s.Priority,
		Attributes: s.Attributes,
		Selector: model.AgentSelector{
			IdentifyingAttributes:    s.Selector.IdentifyingAttributes,
			NonIdentifyingAttributes: s.Selector.NonIdentifyingAttributes,
		},
	}
}

func (s *AgentGroupSpec) toDomain() model.AgentGroupSpec {
	//nolint:exhaustruct // Fields are set conditionally below
	spec := model.AgentGroupSpec{}

	if s.AgentRemoteConfig != nil {
		spec.AgentRemoteConfig = &model.AgentRemoteConfig{
			Value:       []byte(s.AgentRemoteConfig.Value),
			ContentType: s.AgentRemoteConfig.ContentType,
		}
	}

	if s.AgentConnectionConfig != nil {
		spec.AgentConnectionConfig = &model.AgentConnectionConfig{
			OpAMPConnection: model.OpAMPConnectionSettings{
				DestinationEndpoint: s.AgentConnectionConfig.OpAMP.DestinationEndpoint,
				Headers:             s.AgentConnectionConfig.OpAMP.Headers,
				Certificate:         s.AgentConnectionConfig.OpAMP.Certificate.ToDomain(),
			},
			OwnMetrics: model.TelemetryConnectionSettings{
				DestinationEndpoint: s.AgentConnectionConfig.OwnMetrics.DestinationEndpoint,
				Headers:             s.AgentConnectionConfig.OwnMetrics.Headers,
				Certificate:         s.AgentConnectionConfig.OwnMetrics.Certificate.ToDomain(),
			},
			OwnLogs: model.TelemetryConnectionSettings{
				DestinationEndpoint: s.AgentConnectionConfig.OwnLogs.DestinationEndpoint,
				Headers:             s.AgentConnectionConfig.OwnLogs.Headers,
				Certificate:         s.AgentConnectionConfig.OwnLogs.Certificate.ToDomain(),
			},
			OwnTraces: model.TelemetryConnectionSettings{
				DestinationEndpoint: s.AgentConnectionConfig.OwnTraces.DestinationEndpoint,
				Headers:             s.AgentConnectionConfig.OwnTraces.Headers,
				Certificate:         s.AgentConnectionConfig.OwnTraces.Certificate.ToDomain(),
			},
			OtherConnections: lo.MapValues(s.AgentConnectionConfig.OtherConnections,
				func(v ConnectionSettings, _ string) model.OtherConnectionSettings {
					return model.OtherConnectionSettings{
						DestinationEndpoint: v.DestinationEndpoint,
						Headers:             v.Headers,
						Certificate:         v.Certificate.ToDomain(),
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
	return AgentGroupMetadata{
		Name:       metadata.Name,
		Priority:   metadata.Priority,
		Attributes: metadata.Attributes,
		Selector: AgentSelector{
			IdentifyingAttributes:    metadata.Selector.IdentifyingAttributes,
			NonIdentifyingAttributes: metadata.Selector.NonIdentifyingAttributes,
		},
	}
}

func agentGroupSpecFromDomain(spec model.AgentGroupSpec) AgentGroupSpec {
	//nolint:exhaustruct // Fields are set conditionally below
	result := AgentGroupSpec{}

	if spec.AgentRemoteConfig != nil {
		result.AgentRemoteConfig = &AgentRemoteConfig{
			Value:       string(spec.AgentRemoteConfig.Value),
			ContentType: spec.AgentRemoteConfig.ContentType,
		}
	}

	if spec.AgentConnectionConfig != nil {
		result.AgentConnectionConfig = &AgentConnectionConfig{
			OpAMP: ConnectionSettings{
				DestinationEndpoint: spec.AgentConnectionConfig.OpAMPConnection.DestinationEndpoint,
				Headers:             spec.AgentConnectionConfig.OpAMPConnection.Headers,
				Certificate:         NewTelemetryTLSCertificate(spec.AgentConnectionConfig.OpAMPConnection.Certificate),
			},
			OwnMetrics: ConnectionSettings{
				DestinationEndpoint: spec.AgentConnectionConfig.OwnMetrics.DestinationEndpoint,
				Headers:             spec.AgentConnectionConfig.OwnMetrics.Headers,
				Certificate:         NewTelemetryTLSCertificate(spec.AgentConnectionConfig.OwnMetrics.Certificate),
			},
			OwnLogs: ConnectionSettings{
				DestinationEndpoint: spec.AgentConnectionConfig.OwnLogs.DestinationEndpoint,
				Headers:             spec.AgentConnectionConfig.OwnLogs.Headers,
				Certificate:         NewTelemetryTLSCertificate(spec.AgentConnectionConfig.OwnLogs.Certificate),
			},
			OwnTraces: ConnectionSettings{
				DestinationEndpoint: spec.AgentConnectionConfig.OwnTraces.DestinationEndpoint,
				Headers:             spec.AgentConnectionConfig.OwnTraces.Headers,
				Certificate:         NewTelemetryTLSCertificate(spec.AgentConnectionConfig.OwnTraces.Certificate),
			},
			OtherConnections: lo.MapValues(spec.AgentConnectionConfig.OtherConnections,
				func(v model.OtherConnectionSettings, _ string) ConnectionSettings {
					return ConnectionSettings{
						DestinationEndpoint: v.DestinationEndpoint,
						Headers:             v.Headers,
						Certificate:         NewTelemetryTLSCertificate(v.Certificate),
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
