package entity

import (
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	// AgentGroupKeyFieldName is the field name used as the key for AgentGroup entities in MongoDB.
	AgentGroupKeyFieldName string = "name"
)

// AgentGroup is the mongo entity representation of the AgentGroup domain model.
type AgentGroup struct {
	Common   `bson:",inline"`
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

type AgentGroupStatus struct {
	Conditions []Condition `bson:"conditions"`
}

// AgentSelector defines the criteria for selecting agents to be included in the agent group.
type AgentSelector struct {
	IdentifyingAttributes    map[string]string `json:"identifyingAttributes"`
	NonIdentifyingAttributes map[string]string `json:"nonIdentifyingAttributes"`
}

// AgentConfig represents the remote configuration for agents in the group.
type AgentRemoteConfig struct {
	Value       string `bson:"value"       json:"value"`
	ContentType string `bson:"contentType" json:"contentType"`
}

// AgentConnectionConfig represents connection settings for agents in the group.
type AgentConnectionConfig struct {
	OpAMP            ConnectionSettings            `bson:"opamp"  json:"opamp"`
	OwnMetrics       ConnectionSettings            `bson:"ownMetrics"       json:"ownMetrics"`
	OwnLogs          ConnectionSettings            `bson:"ownLogs"          json:"ownLogs"`
	OwnTraces        ConnectionSettings            `bson:"ownTraces"        json:"ownTraces"`
	OtherConnections map[string]ConnectionSettings `bson:"otherConnections" json:"otherConnections"`
}

type ConnectionSettings struct {
	DestinationEndpoint string                   `bson:"destinationEndpoint" json:"destinationEndpoint"`
	Headers             map[string][]string      `bson:"headers,omitempty" json:"headers,omitempty"`
	Certificate         TelemetryTLSCeritificate `bson:"certificate,omitempty" json:"certificate,omitempty"`
}

type TelemetryTLSCeritificate struct {
	Cert       bson.Binary `bson:"cert,omitempty"       json:"cert,omitempty"`
	PrivateKey bson.Binary `bson:"privateKey,omitempty" json:"privateKey,omitempty"`
	CaCert     bson.Binary `bson:"caCert,omitempty"     json:"caCert,omitempty"`
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
	ag := &model.AgentGroup{
		Metadata: e.Metadata.toDomain(),
		Spec:     e.Spec.toDomain(),
		Status:   e.Status.toDomain(),
	}

	if statistics != nil {
		ag.Status.NumAgents = int(statistics.NumAgents)
		ag.Status.NumConnectedAgents = int(statistics.NumConnectedAgents)
		ag.Status.NumHealthyAgents = int(statistics.NumHealthyAgents)
		ag.Status.NumUnhealthyAgents = int(statistics.NumUnhealthyAgents)
		ag.Status.NumNotConnectedAgents = int(statistics.NumNotConnectedAgents)
	}

	return ag
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
				Certificate: model.TelemetryTLSCertificate{
					Cert:       s.AgentConnectionConfig.OpAMP.Certificate.Cert.Data,
					PrivateKey: s.AgentConnectionConfig.OpAMP.Certificate.PrivateKey.Data,
					CaCert:     s.AgentConnectionConfig.OpAMP.Certificate.CaCert.Data,
				},
			},
			OwnMetrics: model.TelemetryConnectionSettings{
				DestinationEndpoint: s.AgentConnectionConfig.OwnMetrics.DestinationEndpoint,
				Headers:             s.AgentConnectionConfig.OwnMetrics.Headers,
				Certificate: model.TelemetryTLSCertificate{
					Cert:       s.AgentConnectionConfig.OwnMetrics.Certificate.Cert.Data,
					PrivateKey: s.AgentConnectionConfig.OwnMetrics.Certificate.PrivateKey.Data,
					CaCert:     s.AgentConnectionConfig.OwnMetrics.Certificate.CaCert.Data,
				},
			},
			OwnLogs: model.TelemetryConnectionSettings{
				DestinationEndpoint: s.AgentConnectionConfig.OwnLogs.DestinationEndpoint,
				Headers:             s.AgentConnectionConfig.OwnLogs.Headers,
				Certificate: model.TelemetryTLSCertificate{
					Cert:       s.AgentConnectionConfig.OwnLogs.Certificate.Cert.Data,
					PrivateKey: s.AgentConnectionConfig.OwnLogs.Certificate.PrivateKey.Data,
					CaCert:     s.AgentConnectionConfig.OwnLogs.Certificate.CaCert.Data,
				},
			},
			OwnTraces: model.TelemetryConnectionSettings{
				DestinationEndpoint: s.AgentConnectionConfig.OwnTraces.DestinationEndpoint,
				Headers:             s.AgentConnectionConfig.OwnTraces.Headers,
				Certificate: model.TelemetryTLSCertificate{
					Cert:       s.AgentConnectionConfig.OwnTraces.Certificate.Cert.Data,
					PrivateKey: s.AgentConnectionConfig.OwnTraces.Certificate.PrivateKey.Data,
					CaCert:     s.AgentConnectionConfig.OwnTraces.Certificate.CaCert.Data,
				},
			},
			OtherConnections: make(map[string]model.OtherConnectionSettings),
		}

		for k, v := range s.AgentConnectionConfig.OtherConnections {
			spec.AgentConnectionConfig.OtherConnections[k] = model.OtherConnectionSettings{
				DestinationEndpoint: v.DestinationEndpoint,
				Headers:             v.Headers,
				Certificate: model.TelemetryTLSCertificate{
					Cert:       v.Certificate.Cert.Data,
					PrivateKey: v.Certificate.PrivateKey.Data,
					CaCert:     v.Certificate.CaCert.Data,
				},
			}
		}
	}

	return spec
}

func (s *AgentGroupStatus) toDomain() model.AgentGroupStatus {
	conditions := make([]model.Condition, len(s.Conditions))
	for i, c := range s.Conditions {
		conditions[i] = model.Condition{
			Type:               model.ConditionType(c.Type),
			LastTransitionTime: c.LastTransitionTime,
			Status:             model.ConditionStatus(c.Status),
			Reason:             c.Reason,
			Message:            c.Message,
		}
	}

	return model.AgentGroupStatus{
		Conditions: conditions,
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
				Certificate: TelemetryTLSCeritificate{
					Cert:       bson.Binary{Data: spec.AgentConnectionConfig.OpAMPConnection.Certificate.Cert},
					PrivateKey: bson.Binary{Data: spec.AgentConnectionConfig.OpAMPConnection.Certificate.PrivateKey},
					CaCert:     bson.Binary{Data: spec.AgentConnectionConfig.OpAMPConnection.Certificate.CaCert},
				},
			},
			OwnMetrics: ConnectionSettings{
				DestinationEndpoint: spec.AgentConnectionConfig.OwnMetrics.DestinationEndpoint,
				Headers:             spec.AgentConnectionConfig.OwnMetrics.Headers,
				Certificate: TelemetryTLSCeritificate{
					Cert:       bson.Binary{Data: spec.AgentConnectionConfig.OwnMetrics.Certificate.Cert},
					PrivateKey: bson.Binary{Data: spec.AgentConnectionConfig.OwnMetrics.Certificate.PrivateKey},
					CaCert:     bson.Binary{Data: spec.AgentConnectionConfig.OwnMetrics.Certificate.CaCert},
				},
			},
			OwnLogs: ConnectionSettings{
				DestinationEndpoint: spec.AgentConnectionConfig.OwnLogs.DestinationEndpoint,
				Headers:             spec.AgentConnectionConfig.OwnLogs.Headers,
				Certificate: TelemetryTLSCeritificate{
					Cert:       bson.Binary{Data: spec.AgentConnectionConfig.OwnLogs.Certificate.Cert},
					PrivateKey: bson.Binary{Data: spec.AgentConnectionConfig.OwnLogs.Certificate.PrivateKey},
					CaCert:     bson.Binary{Data: spec.AgentConnectionConfig.OwnLogs.Certificate.CaCert},
				},
			},
			OwnTraces: ConnectionSettings{
				DestinationEndpoint: spec.AgentConnectionConfig.OwnTraces.DestinationEndpoint,
				Headers:             spec.AgentConnectionConfig.OwnTraces.Headers,
				Certificate: TelemetryTLSCeritificate{
					Cert:       bson.Binary{Data: spec.AgentConnectionConfig.OwnTraces.Certificate.Cert},
					PrivateKey: bson.Binary{Data: spec.AgentConnectionConfig.OwnTraces.Certificate.PrivateKey},
					CaCert:     bson.Binary{Data: spec.AgentConnectionConfig.OwnTraces.Certificate.CaCert},
				},
			},
			OtherConnections: make(map[string]ConnectionSettings),
		}

		for k, v := range spec.AgentConnectionConfig.OtherConnections {
			result.AgentConnectionConfig.OtherConnections[k] = ConnectionSettings{
				DestinationEndpoint: v.DestinationEndpoint,
				Headers:             v.Headers,
				Certificate: TelemetryTLSCeritificate{
					Cert:       bson.Binary{Data: v.Certificate.Cert},
					PrivateKey: bson.Binary{Data: v.Certificate.PrivateKey},
					CaCert:     bson.Binary{Data: v.Certificate.CaCert},
				},
			}
		}
	}

	return result
}

func agentGroupStatusFromDomain(status model.AgentGroupStatus) AgentGroupStatus {
	conditions := make([]Condition, len(status.Conditions))
	for i, c := range status.Conditions {
		conditions[i] = Condition{
			Type:               string(c.Type),
			LastTransitionTime: c.LastTransitionTime,
			Status:             string(c.Status),
			Reason:             c.Reason,
			Message:            c.Message,
		}
	}

	return AgentGroupStatus{
		Conditions: conditions,
	}
}
