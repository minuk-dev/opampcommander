package entity

import (
	"time"

	"github.com/samber/lo"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

const ApplicationKeyFieldName = "metadata.id"

type Application struct {
	Common `bson:",inline"`
	Metadata ApplicationMetadata `bson:"metadata"`
	Spec ApplicationSpec `bson:"spec"`
	Status ApplicationResourceStatus `bson:"status"`
}

type ApplicationMetadata struct {
	ID string `bson:"id"`
	Name string `bson:"name"`
	Labels map[string]string `bson:"labels,omitempty"`
	Annotations map[string]string `bson:"annotations,omitempty"`
	ResourceVersion int64 `bson:"resourceVersion"`
	FirstSeenAt time.Time `bson:"firstSeenAt"`
	LastSeenAt time.Time `bson:"lastSeenAt"`
}

type ApplicationSpec struct {
	Namespace string `bson:"namespace,omitempty"`
	Name string `bson:"name"`
	Versions []string `bson:"versions,omitempty"`
	AgentType string `bson:"agentType,omitempty"`
}

type ApplicationResourceStatus struct {
	AgentInstanceUIDs []string `bson:"agentInstanceUids,omitempty"`
	Conditions []Condition `bson:"conditions,omitempty"`
}

func (a *Application) ToDomain() *agentmodel.Application {
	return &agentmodel.Application{
		Metadata: agentmodel.ApplicationMetadata{ID: a.Metadata.ID, Name: a.Metadata.Name, Labels: a.Metadata.Labels, Annotations: a.Metadata.Annotations, ResourceVersion: a.Metadata.ResourceVersion, FirstSeenAt: a.Metadata.FirstSeenAt, LastSeenAt: a.Metadata.LastSeenAt},
		Spec: agentmodel.ApplicationSpec{Namespace: a.Spec.Namespace, Name: a.Spec.Name, Versions: a.Spec.Versions, AgentType: agent.Type(a.Spec.AgentType)},
		Status: agentmodel.ApplicationStatus{AgentInstanceUIDs: parseUUIDs(a.Status.AgentInstanceUIDs), Conditions: lo.Map(a.Status.Conditions, func(c Condition, _ int) model.Condition { return c.ToDomain() })},
	}
}

func ApplicationFromDomain(a *agentmodel.Application) *Application {
	return &Application{
		Common: Common{Version: VersionV1},
		Metadata: ApplicationMetadata{ID: a.Metadata.ID, Name: a.Metadata.Name, Labels: a.Metadata.Labels, Annotations: a.Metadata.Annotations, ResourceVersion: a.Metadata.ResourceVersion, FirstSeenAt: a.Metadata.FirstSeenAt, LastSeenAt: a.Metadata.LastSeenAt},
		Spec: ApplicationSpec{Namespace: a.Spec.Namespace, Name: a.Spec.Name, Versions: a.Spec.Versions, AgentType: string(a.Spec.AgentType)},
		Status: ApplicationResourceStatus{AgentInstanceUIDs: formatUUIDs(a.Status.AgentInstanceUIDs), Conditions: lo.Map(a.Status.Conditions, func(c model.Condition, _ int) Condition { return NewConditionFromDomain(c) })},
	}
}
