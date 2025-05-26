// Package config provides the configuration for the opampcommander application.
package config

import "log/slog"

// ServerSettings is a struct that holds the server settings.
type ServerSettings struct {
	Addr                string
	EtcdHosts           []string
	LogLevel            slog.Level
	LogFormat           LogFormat
	GithubOAuthSettings *OAuthSettings
}

// LogFormat is a string type that represents the log format.
type LogFormat string

const (
	// LogFormatText represents the text log format.
	LogFormatText LogFormat = "text"
	// LogFormatJSON represents the JSON log format.
	LogFormatJSON LogFormat = "json"
)
