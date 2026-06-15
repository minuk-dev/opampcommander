package host

import (
	"github.com/google/uuid"
	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// mapHostToAPI converts a domain Host to its API representation.
func mapHostToAPI(host *agentmodel.Host) *v1.Host {
	return &v1.Host{
		Kind:       v1.HostKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.HostMetadata{
			ID:          host.Metadata.ID,
			Name:        host.Metadata.Name,
			Labels:      host.Metadata.Labels,
			Annotations: host.Metadata.Annotations,
			FirstSeenAt: v1.NewTime(host.Metadata.FirstSeenAt),
			LastSeenAt:  v1.NewTime(host.Metadata.LastSeenAt),
		},
		Spec: v1.HostSpec{
			Platform:      string(host.Spec.Platform),
			Arch:          host.Spec.Arch,
			Type:          host.Spec.Type,
			OSType:        host.Spec.OS.Type,
			OSVersion:     host.Spec.OS.Version,
			CloudProvider: host.Spec.Cloud.Provider,
			CloudPlatform: host.Spec.Cloud.Platform,
			CloudRegion:   host.Spec.Cloud.Region,
		},
		Status: v1.HostStatus{
			AgentInstanceUIDs: lo.Map(host.Status.AgentInstanceUIDs, func(id uuid.UUID, _ int) string {
				return id.String()
			}),
			Conditions: mapConditionsToAPI(host.Status.Conditions),
		},
	}
}

// mapConditionsToAPI converts domain conditions to their API representation.
func mapConditionsToAPI(conditions []model.Condition) []v1.Condition {
	return lo.Map(conditions, func(cond model.Condition, _ int) v1.Condition {
		return v1.Condition{
			Type:               v1.ConditionType(cond.Type),
			LastTransitionTime: v1.NewTime(cond.LastTransitionTime),
			Status:             v1.ConditionStatus(cond.Status),
			Reason:             cond.Reason,
			Message:            cond.Message,
		}
	})
}
