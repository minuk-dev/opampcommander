// Package config provides the configuration for the opampcommander application.
package config

import (
	"encoding/json"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
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
	RBACModelPath      string
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
