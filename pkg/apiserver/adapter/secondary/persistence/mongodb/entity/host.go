package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model/vo"
)

const (
	// HostKeyFieldName is the key field name for host.
	HostKeyFieldName = "metadata.id"
)

// Host is the MongoDB entity for host.
type Host struct {
	Common `bson:",inline"`

	Metadata HostMetadata       `bson:"metadata"`
	Spec     HostSpec           `bson:"spec"`
	Status   HostResourceStatus `bson:"status"`
}

// HostMetadata represents the metadata of a host.
type HostMetadata struct {
	ID          string            `bson:"id"`
	Name        string            `bson:"name,omitempty"`
	Labels      map[string]string `bson:"labels,omitempty"`
	Annotations map[string]string `bson:"annotations,omitempty"`
	FirstSeenAt time.Time         `bson:"firstSeenAt"`
	LastSeenAt  time.Time         `bson:"lastSeenAt"`
}

// HostSpec represents the spec of a host.
type HostSpec struct {
	Platform      string `bson:"platform"`
	Arch          string `bson:"arch,omitempty"`
	Type          string `bson:"type,omitempty"`
	OSType        string `bson:"osType,omitempty"`
	OSVersion     string `bson:"osVersion,omitempty"`
	CloudProvider string `bson:"cloudProvider,omitempty"`
	CloudPlatform string `bson:"cloudPlatform,omitempty"`
	CloudRegion   string `bson:"cloudRegion,omitempty"`
}

// HostResourceStatus represents the status of a host resource.
type HostResourceStatus struct {
	AgentInstanceUIDs []string    `bson:"agentInstanceUids,omitempty"`
	Conditions        []Condition `bson:"conditions,omitempty"`
}

// ToDomain converts the entity to domain model.
func (h *Host) ToDomain() *agentmodel.Host {
	return &agentmodel.Host{
		Metadata: agentmodel.HostMetadata{
			ID:          h.Metadata.ID,
			Name:        h.Metadata.Name,
			Labels:      h.Metadata.Labels,
			Annotations: h.Metadata.Annotations,
			FirstSeenAt: h.Metadata.FirstSeenAt,
			LastSeenAt:  h.Metadata.LastSeenAt,
		},
		Spec: agentmodel.HostSpec{
			Platform: agent.Platform(h.Spec.Platform),
			Arch:     h.Spec.Arch,
			Type:     h.Spec.Type,
			OS:       vo.OS{Type: h.Spec.OSType, Version: h.Spec.OSVersion},
			Cloud: agent.Cloud{
				Provider: h.Spec.CloudProvider,
				Platform: h.Spec.CloudPlatform,
				Region:   h.Spec.CloudRegion,
			},
		},
		Status: agentmodel.HostStatus{
			AgentInstanceUIDs: parseUUIDs(h.Status.AgentInstanceUIDs),
			Conditions: lo.Map(h.Status.Conditions, func(c Condition, _ int) model.Condition {
				return c.ToDomain()
			}),
		},
	}
}

// HostFromDomain converts domain model to entity.
func HostFromDomain(domain *agentmodel.Host) *Host {
	return &Host{
		Common: Common{
			Version: VersionV1,
			ID:      nil,
		},
		Metadata: HostMetadata{
			ID:          domain.Metadata.ID,
			Name:        domain.Metadata.Name,
			Labels:      domain.Metadata.Labels,
			Annotations: domain.Metadata.Annotations,
			FirstSeenAt: domain.Metadata.FirstSeenAt,
			LastSeenAt:  domain.Metadata.LastSeenAt,
		},
		Spec: HostSpec{
			Platform:      string(domain.Spec.Platform),
			Arch:          domain.Spec.Arch,
			Type:          domain.Spec.Type,
			OSType:        domain.Spec.OS.Type,
			OSVersion:     domain.Spec.OS.Version,
			CloudProvider: domain.Spec.Cloud.Provider,
			CloudPlatform: domain.Spec.Cloud.Platform,
			CloudRegion:   domain.Spec.Cloud.Region,
		},
		Status: HostResourceStatus{
			AgentInstanceUIDs: formatUUIDs(domain.Status.AgentInstanceUIDs),
			Conditions: lo.Map(domain.Status.Conditions, func(c model.Condition, _ int) Condition {
				return NewConditionFromDomain(c)
			}),
		},
	}
}

// parseUUIDs parses a slice of UUID strings, skipping any that are invalid.
func parseUUIDs(ids []string) []uuid.UUID {
	result := make([]uuid.UUID, 0, len(ids))

	for _, id := range ids {
		parsed, err := uuid.Parse(id)
		if err != nil {
			continue
		}

		result = append(result, parsed)
	}

	return result
}

// formatUUIDs renders a slice of UUIDs as their canonical string form.
func formatUUIDs(ids []uuid.UUID) []string {
	return lo.Map(ids, func(id uuid.UUID, _ int) string {
		return id.String()
	})
}
