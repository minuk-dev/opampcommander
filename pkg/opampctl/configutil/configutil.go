// Package configutil provides utilities for working with configuration files in the opampctl tool.
package configutil

import "github.com/minuk-dev/opampcommander/pkg/opampctl/config"

// GetCurrentContext retrieves the current context from the global configuration.
func GetCurrentContext(config *config.GlobalConfig) *config.Context {
	if config == nil || config.CurrentContext == "" {
		return nil
	}

	for _, context := range config.Contexts {
		if context.Name == config.CurrentContext {
			return &context
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

	for _, cluster := range config.Clusters {
		if cluster.Name == currentContext.Cluster {
			return &cluster
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

	for _, user := range config.Users {
		if user.Name == currentContext.User {
			return &user
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
