// Package config provides the configuration for opampctl.
package config

import "path/filepath"

// GlobalConfig contains the global configuration for opampctl.
type GlobalConfig struct {
	// ConfigFilename is the path to the configuration file.
	ConfigFilename string `json:"-" mapstructure:"-" yaml:"-"`

	// CacheDir is the directory where cached files are stored.
	CacheDir string `json:"cacheDir" mapstructure:"cacheDir" yaml:"cacheDir"`

	CurrentContext string    `json:"currentContext" mapstructure:"currentContext" yaml:"currentContext"`
	Contexts       []Context `json:"contexts"       mapstructure:"contexts"       yaml:"contexts"`
	Users          []User    `json:"users"          mapstructure:"users"          yaml:"users"`
	Clusters       []Cluster `json:"clusters"       mapstructure:"clusters"       yaml:"clusters"`
}

// NewDefaultGlobalConfig creates a new GlobalConfig with default values.
func NewDefaultGlobalConfig(homedir string) *GlobalConfig {
	return &GlobalConfig{
		ConfigFilename: filepath.Join(homedir, ".opampcommander", "opampctl", "config.yaml"),
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
	Type string `json:"type" mapstructure:"type" yaml:"type"`
	GithubAuth
	BasicAuth
	ManualAuth
}

// GithubAuth represents the GitHub authentication method for a user in the opampctl configuration.
type GithubAuth struct{}

// BasicAuth represents the basic authentication method for a user in the opampctl configuration.
type BasicAuth struct {
	Username string `json:"username" mapstructure:"username" yaml:"username"`
	Password string `json:"password" mapstructure:"password" yaml:"password"`
}

// ManualAuth represents the manual authentication method for a user in the opampctl configuration.
type ManualAuth struct {
	BearerToken string `json:"bearerToken" mapstructure:"bearerToken" yaml:"bearerToken"`
}
