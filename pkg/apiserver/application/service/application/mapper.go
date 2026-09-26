package application

import (
	"github.com/google/uuid"
	"github.com/samber/lo"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

func mapApplicationToAPI(application *agentmodel.Application) *v1.Application {
	return &v1.Application{
		Kind: v1.ApplicationKind, APIVersion: v1.APIVersion,
		Metadata: v1.ApplicationMetadata{ID: application.Metadata.ID, Name: application.Metadata.Name, Labels: application.Metadata.Labels, Annotations: application.Metadata.Annotations, FirstSeenAt: v1.NewTime(application.Metadata.FirstSeenAt), LastSeenAt: v1.NewTime(application.Metadata.LastSeenAt)},
		Spec: v1.ApplicationSpec{Namespace: application.Spec.Namespace, Name: application.Spec.Name, Versions: application.Spec.Versions, AgentType: string(application.Spec.AgentType)},
		Status: v1.ApplicationStatus{AgentInstanceUIDs: lo.Map(application.Status.AgentInstanceUIDs, func(id uuid.UUID, _ int) string { return id.String() }), Conditions: mapConditionsToAPI(application.Status.Conditions)},
	}
}

func mapConditionsToAPI(conditions []model.Condition) []v1.Condition {
	return lo.Map(conditions, func(condition model.Condition, _ int) v1.Condition {
		return v1.Condition{Type: v1.ConditionType(condition.Type), LastTransitionTime: v1.NewTime(condition.LastTransitionTime), Status: v1.ConditionStatus(condition.Status), Reason: condition.Reason, Message: condition.Message}
	})
}
