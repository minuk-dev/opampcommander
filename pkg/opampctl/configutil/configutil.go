// Package configutil provides utilities for working with configuration files in the opampctl tool.
package configutil

import (
	"log/slog"

	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

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

// GetCurrentCacheDir retrieves the current cacheDir based on the current user from the global configuration.
func GetCurrentCacheDir(config *config.GlobalConfig) string {
	return config.CacheDir
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

// NewLogger creates a new logger for the opampctl tool.
// It uses the default slog logger.
// In future, improve a logger: https://github.com/minuk-dev/opampcommander/issues/53
func NewLogger(*config.GlobalConfig) *slog.Logger {
	return slog.Default()
}
