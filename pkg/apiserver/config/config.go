// Package config provides the configuration for the opampcommander application.
package config

import (
	"encoding/json"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/security"
)

// ServerSettings is a struct that holds the server settings.
//
// It aggregates the per-package configuration owned by the consuming packages
// (security.Config, observability.Config via ManagementSettings, the server
// identity type from the domain) together with the infrastructure settings used
// only by the composition root (database, event, cache).
type ServerSettings struct {
	Address            string
	ServerID           agentmodel.ServerID
	DatabaseSettings   DatabaseSettings
	Security           security.Config
	ManagementSettings ManagementSettings
	EventSettings      EventSettings
	CacheSettings      CacheSettings
	BootstrapSettings  BootstrapSettings
	MetricsBackend     MetricsBackendSettings
	RBACModelPath      string
}

// BootstrapSettings configures how the server seeds built-in resources on startup.
//
// On every start the server reconciles the YAML manifests found under Dir into the
// persistence layer declaratively (full overwrite): the manifests are the source of
// truth, so changes made to built-in resources via the API are reset to match them.
// The default namespace/role names below are the runtime identifiers the server uses
// for agent namespace defaulting and the built-in default-role grant; they should
// match the names declared in the manifests.
type BootstrapSettings struct {
	// Dir is the directory of initial manifest YAML files applied on startup.
	// When empty, manifest reconciliation is skipped (e.g. tests).
	Dir string
	// DefaultNamespace is the namespace an agent is placed in when it does not
	// report a service.namespace identifying attribute. It is also the namespace
	// in which the built-in default role is auto-granted to every user.
	DefaultNamespace string
	// DefaultRole is the name of the built-in role auto-granted to every user.
	DefaultRole string
}

// String returns a JSON representation of the ServerSettings struct.
// It is used for logging and debugging purposes.
//
//nolint:musttag
func (s *ServerSettings) String() string {
	data, err := json.Marshal(s)
	if err != nil {
		return "ServerSettings{error marshaling to JSON}"
	}

	return string(data)
}
