// Package config provides the configuration for opampctl.
package config

import (
	"io"
	"log/slog"
	"path/filepath"
)

// GlobalConfig contains the global configuration for opampctl.
type GlobalConfig struct {
	// CacheDir is the directory where cached files are stored.
	CacheDir string `json:"cacheDir" mapstructure:"cacheDir" yaml:"cacheDir"`

	CurrentContext string    `json:"currentContext" mapstructure:"currentContext" yaml:"currentContext"`
	Contexts       []Context `json:"contexts"       mapstructure:"contexts"       yaml:"contexts"`
	Users          []User    `json:"users"          mapstructure:"users"          yaml:"users"`
	Clusters       []Cluster `json:"clusters"       mapstructure:"clusters"       yaml:"clusters"`

	// Debugging Configuration
	// This configuration is not serialized to the config file.
	Runtime `json:"-" mapstructure:"-" yaml:"-"`
}

// Runtime contains runtime configuration that is not serialized to the config file.
// It's helper to run the command.
type Runtime struct {
	// ConfigFilename is the path to the configuration file.
	ConfigFilename string `json:"-" mapstructure:"-" yaml:"-"`
	// Output is the output writer for the command.
	Output io.Writer `json:"-" mapstructure:"-" yaml:"-"`
	// Log
	Log Log `json:"-" mapstructure:"-" yaml:"-"`
}

// Log contains the logging configuration for opampctl.
type Log struct {
	Logger *slog.Logger
	Level  slog.Level
	Format string
	Writer io.Writer
}

// NewDefaultGlobalConfig creates a new GlobalConfig with default values.
func NewDefaultGlobalConfig(homedir string) *GlobalConfig {
	return &GlobalConfig{
		CurrentContext: "default",
		CacheDir:       filepath.Join(homedir, ".opampcommander", "opampctl", "cache"),
		Contexts: []Context{
			{
				Name:    "default",
				Cluster: "default",
				User:    "default",
			},
		},
		Users: []User{
			{
				Name: "default",
				Auth:
				//exhaustruct:ignore
				Auth{
					Type: "basic",
					BasicAuth: BasicAuth{
						Username: "admin",
						Password: "admin",
					},
				},
			},
		},
		Clusters: []Cluster{
			{
				Name: "default",
				OpAMPCommander: OpAMPCommander{
					Endpoint: "http://localhost:8080",
				},
			},
		},
		Runtime: Runtime{
			ConfigFilename: filepath.Join(homedir, ".opampcommander", "opampctl", "config.yaml"),
			Output:         io.Discard, // Default output is discarded
			Log: Log{
				Logger: nil,
				Level:  slog.LevelInfo,
				Format: "text",
				Writer: io.Discard, // Default log writer is discarded
			},
		},
	}
}

// Context represents a context in the opampctl configuration.
type Context struct {
	Name    string `json:"name"    mapstructure:"name"    yaml:"name"`
	Cluster string `json:"cluster" mapstructure:"cluster" yaml:"cluster"`
	User    string `json:"user"    mapstructure:"user"    yaml:"user"`
}

// Cluster represents a cluster in the opampctl configuration.
type Cluster struct {
	Name           string         `json:"name"           mapstructure:"name"           yaml:"name"`
	OpAMPCommander OpAMPCommander `json:"opampcommander" mapstructure:"opampcommander" yaml:"opampcommander"`
}

// OpAMPCommander represents the OpAMP Commander configuration in the opampctl configuration.
type OpAMPCommander struct {
	Endpoint string `json:"endpoint" mapstructure:"endpoint" yaml:"endpoint"`
}

// User represents a user in the opampctl configuration.
type User struct {
	Name string `json:"name" mapstructure:"name" yaml:"name"`
	Auth Auth   `json:"auth" mapstructure:"auth" yaml:"auth"`
}

const (
	// AuthTypeGithub indicates that the user is authenticated using GitHub.
	AuthTypeGithub = "github"
	// AuthTypeBasic indicates that the user is authenticated using basic authentication.
	AuthTypeBasic = "basic"
	// AuthTypeManual indicates that the user is authenticated using a manual method (e.g., bearer token).
	AuthTypeManual = "manual"
)

// Auth represents the authentication method for a user in the opampctl configuration.
type Auth struct {
	Type       string `json:"type"    mapstructure:"type"    yaml:"type"`
	GithubAuth `json:",inline" mapstructure:",squash" yaml:",inline"`
	BasicAuth  `json:",inline" mapstructure:",squash" yaml:",inline"`
	ManualAuth `json:",inline" mapstructure:",squash" yaml:",inline"`
}

// GithubAuth represents the GitHub authentication method for a user in the opampctl configuration.
type GithubAuth struct{}

// BasicAuth represents the basic authentication method for a user in the opampctl configuration.
type BasicAuth struct {
	Username string `json:"username,omitempty" mapstructure:"username,omitempty" yaml:"username,omitempty"`
	Password string `json:"password,omitempty" mapstructure:"password,omitempty" yaml:"password,omitempty"`
}

// ManualAuth represents the manual authentication method for a user in the opampctl configuration.
type ManualAuth struct {
	BearerToken string `json:"bearerToken,omitempty" mapstructure:"bearerToken,omitempty" yaml:"bearerToken,omitempty"`
}
