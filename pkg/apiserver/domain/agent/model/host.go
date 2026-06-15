package agentmodel

import (
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model/vo"
)

// Host is a domain aggregate that represents the machine (bare metal or VM) one
// or more agents run on. Hosts are not created by users; they are discovered
// and upserted from the OpenTelemetry "host.*"/"cloud.*" attributes an agent
// reports in its description.
type Host struct {
	Metadata HostMetadata
	Spec     HostSpec
	Status   HostStatus
}

// HostMetadata holds identity and lifecycle information for a host.
type HostMetadata struct {
	// ID is the stable identity of the host: the OpenTelemetry "host.id"
	// attribute, falling back to "host.name" when "host.id" is absent.
	ID string
	// Name is the reported "host.name".
	Name string
	// Labels and Annotations are reserved for user-supplied metadata.
	Labels      map[string]string
	Annotations map[string]string
	// FirstSeenAt is when the host was first discovered.
	FirstSeenAt time.Time
	// LastSeenAt is the most recent time an agent on this host reported.
	LastSeenAt time.Time
}

// HostSpec holds the discovered, descriptive facts about a host.
type HostSpec struct {
	// Platform classifies the deployment environment (baremetal, vm, ...).
	Platform agent.Platform
	// Arch is the reported "host.arch".
	Arch string
	// Type is the reported "host.type" (e.g. a cloud instance type).
	Type string
	// OS is the reported operating system.
	OS vo.OS
	// Cloud is the reported cloud context, if any.
	Cloud agent.Cloud
}

// HostStatus holds the observed state of a host.
type HostStatus struct {
	// AgentInstanceUIDs are the agents currently associated with this host.
	AgentInstanceUIDs []uuid.UUID
	// Conditions is a list of conditions that apply to the host.
	Conditions []model.Condition
}

// HostIDOf returns the stable identity for the host described by desc: the
// "host.id" attribute, falling back to "host.name". It returns an empty string
// when the description carries no host attributes.
func HostIDOf(desc agent.Description) string {
	host := desc.Host()
	if host.ID != "" {
		return host.ID
	}

	return host.Name
}

// NewHost creates a new, empty host with the given identity. Use ObserveAgent to
// populate it from an agent's reported description.
func NewHost(id string, now time.Time) *Host {
	return &Host{
		Metadata: HostMetadata{
			ID:          id,
			Name:        "",
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
			FirstSeenAt: now,
			LastSeenAt:  now,
		},
		Spec: HostSpec{
			Platform: agent.PlatformUnknown,
			Arch:     "",
			Type:     "",
			OS:       vo.OS{Type: "", Version: ""},
			Cloud:    agent.Cloud{Provider: "", Platform: "", Region: ""},
		},
		Status: HostStatus{
			AgentInstanceUIDs: nil,
			Conditions:        nil,
		},
	}
}

// ObserveAgent refreshes the host's discovered spec from an agent's description,
// advances LastSeenAt, and ensures the agent is associated with this host.
func (h *Host) ObserveAgent(instanceUID uuid.UUID, desc agent.Description, now time.Time) {
	host := desc.Host()
	if host.Name != "" {
		h.Metadata.Name = host.Name
	}

	h.Spec.Platform = desc.Platform()
	h.Spec.Arch = host.Arch
	h.Spec.Type = host.Type
	h.Spec.OS = desc.OS()
	h.Spec.Cloud = desc.Cloud()

	h.Metadata.LastSeenAt = now
	h.Status.AgentInstanceUIDs = appendUniqueUID(h.Status.AgentInstanceUIDs, instanceUID)
}

// appendUniqueUID appends uid to uids only if it is not already present.
func appendUniqueUID(uids []uuid.UUID, uid uuid.UUID) []uuid.UUID {
	if slices.Contains(uids, uid) {
		return uids
	}

	return append(uids, uid)
}
