package agentmodel

import (
	"encoding/base64"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// Application is a discovered logical service, keyed by service.namespace and
// service.name rather than by an individual service instance.
type Application struct {
	Metadata ApplicationMetadata
	Spec     ApplicationSpec
	Status   ApplicationStatus
}

type ApplicationMetadata struct {
	ID              string
	Name            string
	Labels          map[string]string
	Annotations     map[string]string
	ResourceVersion int64
	FirstSeenAt     time.Time
	LastSeenAt      time.Time
}

// ApplicationSpec holds the service attributes reported by its agents.
type ApplicationSpec struct {
	Namespace string
	Name      string
	Versions  []string
	AgentType agent.Type
}

type ApplicationStatus struct {
	AgentInstanceUIDs []uuid.UUID
	Conditions        []model.Condition
}

// ApplicationIDOf returns a URL-safe, reversible composite identity for the
// service namespace and name. A service without a name cannot be discovered.
func ApplicationIDOf(desc agent.Description) string {
	service := desc.Service()
	if service.Name == "" {
		return ""
	}

	return base64.RawURLEncoding.EncodeToString([]byte(service.Namespace + "\x00" + service.Name))
}

func NewApplication(id string, now time.Time) *Application {
	return &Application{
		Metadata: ApplicationMetadata{
			ID: id, Labels: make(map[string]string), Annotations: make(map[string]string),
			FirstSeenAt: now, LastSeenAt: now,
		},
	}
}

// ObserveAgent updates observed service facts and associates the agent.
func (a *Application) ObserveAgent(instanceUID uuid.UUID, desc agent.Description, now time.Time) {
	service := desc.Service()
	a.Metadata.Name = service.Name
	a.Spec.Namespace = service.Namespace
	a.Spec.Name = service.Name
	a.Spec.AgentType = desc.AgentType()
	if service.Version != "" && !slices.Contains(a.Spec.Versions, service.Version) {
		a.Spec.Versions = append(a.Spec.Versions, service.Version)
	}
	a.Metadata.LastSeenAt = now
	if !slices.Contains(a.Status.AgentInstanceUIDs, instanceUID) {
		a.Status.AgentInstanceUIDs = append(a.Status.AgentInstanceUIDs, instanceUID)
	}
}
