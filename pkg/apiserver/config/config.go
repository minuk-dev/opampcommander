// Package config provides the configuration for the opampcommander application.
package config

import (
	"encoding/json"
)

// ServerID is a unique identifier for an API server instance.
type ServerID string

// String returns the string representation of the ServerID.
func (s ServerID) String() string {
	return string(s)
}

// ServerSettings is a struct that holds the server settings.
type ServerSettings struct {
	Address            string
	ServerID           ServerID
	DatabaseSettings   DatabaseSettings
	AuthSettings       AuthSettings
	ManagementSettings ManagementSettings
	EventSettings      EventSettings
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
