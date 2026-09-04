package port

import (
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	usermodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user"
)

// Field-selector allowlists, re-exported per resource for the primary adapters.
//
// A controller has to reject an unsupported fieldSelector field with a 400 that
// names it, which means knowing the allowlist — but primary adapters may not
// import the domain. The lists are the domain's, surfaced here rather than
// copied, so there is one definition to keep in step with the storage schema.
//
//nolint:gochecknoglobals // re-export of the domain's declarative schema
var (
	// AgentSelectableFields are the fields an agent listing can be filtered on.
	AgentSelectableFields = agentmodel.AgentSelectableFields
	// AgentGroupSelectableFields are the fields an agent-group listing can be filtered on.
	AgentGroupSelectableFields = agentmodel.AgentGroupSelectableFields
	// AgentPackageSelectableFields are the fields an agent-package listing can be filtered on.
	AgentPackageSelectableFields = agentmodel.AgentPackageSelectableFields
	// AgentRemoteConfigSelectableFields are the fields a remote-config listing can be filtered on.
	AgentRemoteConfigSelectableFields = agentmodel.AgentRemoteConfigSelectableFields
	// CertificateSelectableFields are the fields a certificate listing can be filtered on.
	CertificateSelectableFields = agentmodel.CertificateSelectableFields
	// ContainerSelectableFields are the fields a container listing can be filtered on.
	ContainerSelectableFields = agentmodel.ContainerSelectableFields
	// EndpointSelectableFields are the fields an endpoint listing can be filtered on.
	EndpointSelectableFields = agentmodel.EndpointSelectableFields
	// HostSelectableFields are the fields a host listing can be filtered on.
	HostSelectableFields = agentmodel.HostSelectableFields
	// NamespaceSelectableFields are the fields a namespace listing can be filtered on.
	NamespaceSelectableFields = agentmodel.NamespaceSelectableFields
	// RemoteConfigSchemaSelectableFields are the fields a schema listing can be filtered on.
	RemoteConfigSchemaSelectableFields = agentmodel.RemoteConfigSchemaSelectableFields
	// UserSelectableFields are the fields a user listing can be filtered on.
	UserSelectableFields = usermodel.UserSelectableFields
)
