// Package config provides the configuration for opampctl.
package config

// GlobalConfig contains the global configuration for opampctl.
type GlobalConfig struct {
	CurrentContext string    `mapstructure:"currentContext"`
	Contexts       []Context `mapstructure:"contexts"`
	Users          []User    `mapstructure:"users"`
	Clusters       []Cluster `mapstructure:"clusters"`
}

// Context represents a context in the opampctl configuration.
type Context struct {
	Name    string `mapstructure:"name"`
	Cluster string `mapstructure:"cluster"`
	User    string `mapstructure:"user"`
}

// Cluster represents a cluster in the opampctl configuration.
type Cluster struct {
	Name           string         `mapstructure:"name"`
	OpAMPCommander OpAMPCommander `mapstructure:"opampcommander"`
}

// OpAMPCommander represents the OpAMP Commander configuration in the opampctl configuration.
type OpAMPCommander struct {
	Endpoint string `mapstructure:"endpoint"`
}

// User represents a user in the opampctl configuration.
type User struct {
	Name string `mapstructure:"name"`
	Auth Auth   `mapstructure:"auth"`
}

// Auth represents the authentication method for a user in the opampctl configuration.
type Auth struct {
	Type string `mapstructure:"type"`
	GithubAuth
	BasicAuth
	ManualAuth
}

// GithubAuth represents the GitHub authentication method for a user in the opampctl configuration.
type GithubAuth struct{}

// BasicAuth represents the basic authentication method for a user in the opampctl configuration.
type BasicAuth struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// ManualAuth represents the manual authentication method for a user in the opampctl configuration.
type ManualAuth struct {
	BearerToken string `mapstructure:"bearerToken"`
}
