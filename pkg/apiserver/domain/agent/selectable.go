package agentmodel

import (
	"strconv"
	"time"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// Field-selector allowlists, one per aggregate.
//
// A field selector may only reference a field listed here: the API boundary
// rejects anything else with a 400 naming the field, so a client never silently
// receives an unfiltered list. Every listed field is backed by an index, and by
// an entry in the MongoDB adapter's translation table — a parity test enforces
// both directions, so adding a field here without indexing and translating it
// fails the build rather than silently degrading into a collection scan.
//
//nolint:gochecknoglobals // declarative per-aggregate schema, read-only after init
var (
	// AgentSelectableFields are the fields an agent listing can be filtered on.
	AgentSelectableFields = []string{
		"metadata.namespace",
		"status.connected",
		"status.healthy",
	}
	// AgentGroupSelectableFields are the fields an agent-group listing can be filtered on.
	AgentGroupSelectableFields = []string{
		"metadata.namespace",
	}
	// AgentPackageSelectableFields are the fields an agent-package listing can be filtered on.
	AgentPackageSelectableFields = []string{
		"metadata.namespace",
		"spec.packageType",
		"spec.version",
	}
	// AgentRemoteConfigSelectableFields are the fields a remote-config listing can be filtered on.
	AgentRemoteConfigSelectableFields = []string{
		"metadata.namespace",
	}
	// CertificateSelectableFields are the fields a certificate listing can be filtered on.
	CertificateSelectableFields = []string{
		"metadata.namespace",
	}
	// ContainerSelectableFields are the fields a container listing can be filtered on.
	ContainerSelectableFields = []string{
		"spec.platform",
	}
	// EndpointSelectableFields are the fields an endpoint listing can be filtered on.
	EndpointSelectableFields = []string{
		"metadata.namespace",
		"spec.protocol",
	}
	// HostSelectableFields are the fields a host listing can be filtered on.
	HostSelectableFields = []string{
		"spec.platform",
	}
	// NamespaceSelectableFields are the fields a namespace listing can be filtered
	// on. A namespace is little more than its name, so that is the one field —
	// the exact-match complement to the "?name=" prefix search.
	NamespaceSelectableFields = []string{
		"metadata.name",
	}
	// RemoteConfigSchemaSelectableFields are the fields a schema listing can be filtered on.
	RemoteConfigSchemaSelectableFields = []string{
		"metadata.namespace",
		"spec.binary",
		"spec.version",
	}
)

// SelectorValuesAt returns the projection server-side selectors filter agents on.
//
// Connectedness is evaluated at now, exactly as [Agent.IsConnectedAt] does, so a
// "status.connected=true" field selector and the connected badge a client renders
// cannot disagree.
//
// An agent's labels are its identifying attributes. An agent carries no
// user-supplied label map — the OpAMP protocol gives an operator the agent's
// description instead, which is already what agent groups select on — so that is
// what a label selector reads.
func (a *Agent) SelectorValuesAt(now time.Time) model.SelectorValues {
	return model.SelectorValues{
		Name:             a.Metadata.InstanceUID.String(),
		Labels:           a.Metadata.Description.IdentifyingAttributes,
		AdditionalLabels: a.Metadata.Description.NonIdentifyingAttributes,
		Fields: map[string]string{
			"metadata.namespace": a.Metadata.Namespace,
			"status.connected":   strconv.FormatBool(a.IsConnectedAt(now, DefaultConnectionStaleness)),
			"status.healthy":     strconv.FormatBool(a.Status.ComponentHealth.Healthy),
		},
	}
}

// SelectorValues returns the projection server-side selectors filter agent
// groups on.
func (a *AgentGroup) SelectorValues() model.SelectorValues {
	return model.SelectorValues{
		Name:             a.Metadata.Name,
		Labels:           a.Metadata.Attributes,
		AdditionalLabels: nil,
		Fields: map[string]string{
			"metadata.namespace": a.Metadata.Namespace,
		},
	}
}

// SelectorValues returns the projection server-side selectors filter agent
// packages on.
func (a *AgentPackage) SelectorValues() model.SelectorValues {
	return model.SelectorValues{
		Name:             a.Metadata.Name,
		Labels:           a.Metadata.Attributes,
		AdditionalLabels: nil,
		Fields: map[string]string{
			"metadata.namespace": a.Metadata.Namespace,
			"spec.packageType":   a.Spec.PackageType,
			"spec.version":       a.Spec.Version,
		},
	}
}

// SelectorValues returns the projection server-side selectors filter agent
// remote configs on.
func (a *AgentRemoteConfig) SelectorValues() model.SelectorValues {
	return model.SelectorValues{
		Name:             a.Metadata.Name,
		Labels:           a.Metadata.Attributes,
		AdditionalLabels: nil,
		Fields: map[string]string{
			"metadata.namespace": a.Metadata.Namespace,
		},
	}
}

// SelectorValues returns the projection server-side selectors filter
// certificates on.
func (c *Certificate) SelectorValues() model.SelectorValues {
	return model.SelectorValues{
		Name:             c.Metadata.Name,
		Labels:           c.Metadata.Attributes,
		AdditionalLabels: nil,
		Fields: map[string]string{
			"metadata.namespace": c.Metadata.Namespace,
		},
	}
}

// SelectorValues returns the projection server-side selectors filter containers
// on.
func (c *Container) SelectorValues() model.SelectorValues {
	return model.SelectorValues{
		Name:             c.Metadata.Name,
		Labels:           c.Metadata.Labels,
		AdditionalLabels: nil,
		Fields: map[string]string{
			"spec.platform": string(c.Spec.Platform),
		},
	}
}

// SelectorValues returns the projection server-side selectors filter endpoints
// on.
func (e *Endpoint) SelectorValues() model.SelectorValues {
	return model.SelectorValues{
		Name:             e.Metadata.Name,
		Labels:           e.Metadata.Attributes,
		AdditionalLabels: nil,
		Fields: map[string]string{
			"metadata.namespace": e.Metadata.Namespace,
			"spec.protocol":      e.Spec.Protocol,
		},
	}
}

// SelectorValues returns the projection server-side selectors filter hosts on.
func (h *Host) SelectorValues() model.SelectorValues {
	return model.SelectorValues{
		Name:             h.Metadata.Name,
		Labels:           h.Metadata.Labels,
		AdditionalLabels: nil,
		Fields: map[string]string{
			"spec.platform": string(h.Spec.Platform),
		},
	}
}

// SelectorValues returns the projection server-side selectors filter namespaces
// on.
func (n *Namespace) SelectorValues() model.SelectorValues {
	return model.SelectorValues{
		Name:             n.Metadata.Name,
		Labels:           n.Metadata.Labels,
		AdditionalLabels: nil,
		Fields: map[string]string{
			"metadata.name": n.Metadata.Name,
		},
	}
}

// SelectorValues returns the projection server-side selectors filter remote
// config schemas on.
func (r *RemoteConfigSchema) SelectorValues() model.SelectorValues {
	return model.SelectorValues{
		Name:             r.Metadata.Name,
		Labels:           r.Metadata.Attributes,
		AdditionalLabels: nil,
		Fields: map[string]string{
			"metadata.namespace": r.Metadata.Namespace,
			"spec.binary":        r.Spec.Binary,
			"spec.version":       r.Spec.Version,
		},
	}
}
