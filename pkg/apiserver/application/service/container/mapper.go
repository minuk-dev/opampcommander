package container

import (
	"github.com/google/uuid"
	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// mapContainerToAPI converts a domain Container to its API representation.
func mapContainerToAPI(container *agentmodel.Container) *v1.Container {
	return &v1.Container{
		Kind:       v1.ContainerKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.ContainerMetadata{
			ID:          container.Metadata.ID,
			Name:        container.Metadata.Name,
			Labels:      container.Metadata.Labels,
			Annotations: container.Metadata.Annotations,
			FirstSeenAt: v1.NewTime(container.Metadata.FirstSeenAt),
			LastSeenAt:  v1.NewTime(container.Metadata.LastSeenAt),
		},
		Spec: v1.ContainerSpec{
			Platform:         string(container.Spec.Platform),
			ImageName:        container.Spec.ImageName,
			Runtime:          container.Spec.Runtime,
			HostID:           container.Spec.HostID,
			K8sPodName:       container.Spec.K8s.PodName,
			K8sNamespaceName: container.Spec.K8s.NamespaceName,
			K8sNodeName:      container.Spec.K8s.NodeName,
		},
		Status: v1.ContainerStatus{
			AgentInstanceUIDs: lo.Map(container.Status.AgentInstanceUIDs, func(id uuid.UUID, _ int) string {
				return id.String()
			}),
			Conditions: mapConditionsToAPI(container.Status.Conditions),
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
