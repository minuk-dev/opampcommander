// Package configutil provides utilities for working with configuration files in the opampctl tool.
package configutil

import "github.com/minuk-dev/opampcommander/pkg/opampctl/config"

// GetCurrentContext retrieves the current context from the global configuration.
func GetCurrentContext(config *config.GlobalConfig) *config.Context {
	if config == nil || config.CurrentContext == "" {
		return nil
	}

	for i := range config.Contexts {
		if config.Contexts[i].Name == config.CurrentContext {
			return &config.Contexts[i]
		}
	}

	return nil
}

// GetCurrentCluster retrieves the current cluster based on the current context from the global configuration.
func GetCurrentCluster(config *config.GlobalConfig) *config.Cluster {
	if config == nil || config.CurrentContext == "" {
		return nil
	}

	currentContext := GetCurrentContext(config)
	if currentContext == nil {
		return nil
	}

	for i := range config.Clusters {
		if config.Clusters[i].Name == currentContext.Cluster {
			return &config.Clusters[i]
		}
	}

	return nil
}

// GetCurrentUser retrieves the current user based on the current context from the global configuration.
func GetCurrentUser(config *config.GlobalConfig) *config.User {
	if config == nil || config.CurrentContext == "" {
		return nil
	}

	currentContext := GetCurrentContext(config)
	if currentContext == nil {
		return nil
	}

	for i := range config.Users {
		if config.Users[i].Name == currentContext.User {
			return &config.Users[i]
		}
	}

	return nil
}

// GetCurrentOpAMPCommanderEndpoint retrieves the OpAMP Commander endpoint for the current cluster
// from the global configuration.
func GetCurrentOpAMPCommanderEndpoint(config *config.GlobalConfig) string {
	currentCluster := GetCurrentCluster(config)
	if currentCluster == nil {
		return ""
	}

	return currentCluster.OpAMPCommander.Endpoint
}
