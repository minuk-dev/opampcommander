// Package config provides the configuration for the opampcommander application.
package config

import (
	"encoding/json"
	"log/slog"
)

// ServerSettings is a struct that holds the server settings.
type ServerSettings struct {
	Address           string
	DatabaseEndpoints []string
	LogLevel          slog.Level
	LogFormat         LogFormat
	AuthSettings      *AuthSettings
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

// LogFormat is a string type that represents the log format.
type LogFormat string

const (
	// LogFormatText represents the text log format.
	LogFormatText LogFormat = "text"
	// LogFormatJSON represents the JSON log format.
	LogFormatJSON LogFormat = "json"
)
