// Package config provides the configuration for the opampcommander application.
package config

import (
	"encoding/json"
)

// ServerSettings is a struct that holds the server settings.
type ServerSettings struct {
	Address                string
	DatabaseSettings       DatabaseSettings
	AuthSettings           AuthSettings
	ObservabiilitySettings ObservabilitySettings
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
