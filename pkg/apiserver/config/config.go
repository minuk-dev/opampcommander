// Package config provides the configuration for the opampcommander application.
package config

import (
	"encoding/json"

	runtimeapi "github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/primary/http/runtime"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
)

// ServerSettings is a struct that holds the server settings.
//
// It aggregates the per-package configuration owned by the consuming packages
// (observability.Config via ManagementSettings, the server
// identity type from the domain) together with the infrastructure settings used
// only by the composition root (database, event, cache).
type ServerSettings struct {
	RuntimeAPI         runtimeapi.Config
	Address            string
	Shutdown           ShutdownSettings
	TLS                TLSSettings
	OpAMP              OpAMPSettings
	OpAMPTLS           OpAMPTLSSettings
	ServerID           agentmodel.ServerID
	DatabaseSettings   DatabaseSettings
	ManagementSettings ManagementSettings
	EventSettings      EventSettings
	CacheSettings      CacheSettings
	LivenessSettings   LivenessSettings
	BootstrapSettings  BootstrapSettings
	MetricsBackend     MetricsBackendSettings
}

// OpAMPTLSSettings enables HTTPS and client certificate authentication on the
// OpAMP endpoint. All three paths are required when enabled.
type OpAMPTLSSettings struct {
	CertFile string
	KeyFile  string
	CAFile   string
}

// RemoteConfigSchema load policies for BootstrapSettings.RemoteConfigSchemaLoad.
const (
	// RemoteConfigSchemaLoadLatest seeds only the newest version per distribution.
	RemoteConfigSchemaLoadLatest = "latest"
	// RemoteConfigSchemaLoadAll seeds every version found in the schema library.
	RemoteConfigSchemaLoadAll = "all"
	// RemoteConfigSchemaLoadNone disables schema seeding.
	RemoteConfigSchemaLoadNone = "none"
)

// BootstrapSettings configures how the server seeds built-in resources on startup.
//
// On every start the server reconciles the YAML manifests found under Dir into the
// persistence layer declaratively (full overwrite): the manifests are the source of
// truth, so changes made to built-in resources via the API are reset to match them.
// DefaultNamespace should match a namespace declared in the manifests.
type BootstrapSettings struct {
	// Dir is the directory of initial manifest YAML files applied on startup.
	// When empty, manifest reconciliation is skipped (e.g. tests).
	Dir string
	// RemoteConfigSchemaDir is the directory of the pre-built RemoteConfigSchema
	// library (one YAML per collector distribution/version). When empty it defaults
	// to the "remoteconfigschema" subdirectory of Dir; when both are empty, schema
	// seeding is skipped. The main manifest reconciler ignores this subdirectory, so
	// it is loaded separately according to RemoteConfigSchemaLoad.
	RemoteConfigSchemaDir string
	// RemoteConfigSchemaLoad selects which schemas from RemoteConfigSchemaDir are
	// seeded on startup: "latest" (default) seeds only the newest version per
	// distribution, "all" seeds every version in the library, and "none" disables
	// schema seeding while keeping the library on disk for opt-in use.
	RemoteConfigSchemaLoad string
	// DefaultNamespace is the namespace an agent is placed in when it does not
	// report a service.namespace identifying attribute.
	DefaultNamespace string
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
