package entity

import (
	"time"

	"github.com/samber/lo"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

const (
	// ContainerKeyFieldName is the key field name for container.
	ContainerKeyFieldName = "metadata.id"
)

// Container is the MongoDB entity for container.
type Container struct {
	Common `bson:",inline"`

	Metadata ContainerMetadata       `bson:"metadata"`
	Spec     ContainerSpec           `bson:"spec"`
	Status   ContainerResourceStatus `bson:"status"`
}

// ContainerMetadata represents the metadata of a container.
type ContainerMetadata struct {
	ID          string            `bson:"id"`
	Name        string            `bson:"name,omitempty"`
	Labels      map[string]string `bson:"labels,omitempty"`
	Annotations map[string]string `bson:"annotations,omitempty"`
	FirstSeenAt time.Time         `bson:"firstSeenAt"`
	LastSeenAt  time.Time         `bson:"lastSeenAt"`
}

// ContainerSpec represents the spec of a container.
type ContainerSpec struct {
	Platform         string `bson:"platform"`
	ImageName        string `bson:"imageName,omitempty"`
	Runtime          string `bson:"runtime,omitempty"`
	HostID           string `bson:"hostId,omitempty"`
	K8sPodName       string `bson:"k8sPodName,omitempty"`
	K8sPodUID        string `bson:"k8sPodUid,omitempty"`
	K8sNamespaceName string `bson:"k8sNamespaceName,omitempty"`
	K8sNodeName      string `bson:"k8sNodeName,omitempty"`
	K8sContainerName string `bson:"k8sContainerName,omitempty"`
}

// ContainerResourceStatus represents the status of a container resource.
type ContainerResourceStatus struct {
	AgentInstanceUIDs []string    `bson:"agentInstanceUids,omitempty"`
	Conditions        []Condition `bson:"conditions,omitempty"`
}

// ToDomain converts the entity to domain model.
func (c *Container) ToDomain() *agentmodel.Container {
	return &agentmodel.Container{
		Metadata: agentmodel.ContainerMetadata{
			ID:          c.Metadata.ID,
			Name:        c.Metadata.Name,
			Labels:      c.Metadata.Labels,
			Annotations: c.Metadata.Annotations,
			FirstSeenAt: c.Metadata.FirstSeenAt,
			LastSeenAt:  c.Metadata.LastSeenAt,
		},
		Spec: agentmodel.ContainerSpec{
			Platform:  agent.Platform(c.Spec.Platform),
			ImageName: c.Spec.ImageName,
			Runtime:   c.Spec.Runtime,
			HostID:    c.Spec.HostID,
			K8s: agent.K8s{
				PodName:       c.Spec.K8sPodName,
				PodUID:        c.Spec.K8sPodUID,
				NamespaceName: c.Spec.K8sNamespaceName,
				NodeName:      c.Spec.K8sNodeName,
				ContainerName: c.Spec.K8sContainerName,
			},
		},
		Status: agentmodel.ContainerStatus{
			AgentInstanceUIDs: parseUUIDs(c.Status.AgentInstanceUIDs),
			Conditions: lo.Map(c.Status.Conditions, func(cond Condition, _ int) model.Condition {
				return cond.ToDomain()
			}),
		},
	}
}

// ContainerFromDomain converts domain model to entity.
func ContainerFromDomain(domain *agentmodel.Container) *Container {
	return &Container{
		Common: Common{
			Version: VersionV1,
			ID:      nil,
		},
		Metadata: ContainerMetadata{
			ID:          domain.Metadata.ID,
			Name:        domain.Metadata.Name,
			Labels:      domain.Metadata.Labels,
			Annotations: domain.Metadata.Annotations,
			FirstSeenAt: domain.Metadata.FirstSeenAt,
			LastSeenAt:  domain.Metadata.LastSeenAt,
		},
		Spec: ContainerSpec{
			Platform:         string(domain.Spec.Platform),
			ImageName:        domain.Spec.ImageName,
			Runtime:          domain.Spec.Runtime,
			HostID:           domain.Spec.HostID,
			K8sPodName:       domain.Spec.K8s.PodName,
			K8sPodUID:        domain.Spec.K8s.PodUID,
			K8sNamespaceName: domain.Spec.K8s.NamespaceName,
			K8sNodeName:      domain.Spec.K8s.NodeName,
			K8sContainerName: domain.Spec.K8s.ContainerName,
		},
		Status: ContainerResourceStatus{
			AgentInstanceUIDs: formatUUIDs(domain.Status.AgentInstanceUIDs),
			Conditions: lo.Map(domain.Status.Conditions, func(c model.Condition, _ int) Condition {
				return NewConditionFromDomain(c)
			}),
		},
	}
}
