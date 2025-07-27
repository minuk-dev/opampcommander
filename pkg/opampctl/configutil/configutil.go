// Package configutil provides utilities for working with configuration files in the opampctl tool.
package configutil

import (
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

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

// CreateGlobalConfigFlags creates flags for the global configuration.
func CreateGlobalConfigFlags(flags *pflag.FlagSet) {
	flags.StringP("config", "c", "",
		`Path to the configuration file (yaml format).
If not specified, it will look for a config file in the default location:
$HOME/.config/opampcommander/opampctl/config.yaml`)
	flags.BoolP("verbose", "v", false, "Enable verbose output. Equivalent to --log.level=debug")
	flags.String("log.format", "text", "Log output format (text, json)")
	flags.String("log.level", "info", "Log level (debug, info, warn)")
}

// ApplyCmdFlags applies the command flags to the global configuration and returns the updated configuration.
func ApplyCmdFlags(globalConfig *config.GlobalConfig, cmd *cobra.Command) (*config.GlobalConfig, error) {
	flags := cmd.Flags()

	verbose, err := flags.GetBool("verbose")
	if err != nil {
		return nil, fmt.Errorf("failed to get verbose flag: %w", err)
	}

	logFormat, err := flags.GetString("log.format")
	if err != nil {
		return nil, fmt.Errorf("failed to get log format: %w", err)
	}

	logLevel, err := flags.GetString("log.level")
	if err != nil {
		return nil, fmt.Errorf("failed to get log level: %w", err)
	}

	if verbose {
		logLevel = "debug"
	}

	var level slog.Level

	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	globalConfig.Log = config.Log{
		Logger: newLogger(globalConfig),
		Format: logFormat,
		Level:  level,
		Writer: cmd.ErrOrStderr(),
	}
	globalConfig.Output = cmd.OutOrStdout()

	return globalConfig, nil
}

// GetLogger creates a new logger for the opampctl tool.
// It uses the default slog logger.
// In future, improve a logger: https://github.com/minuk-dev/opampcommander/issues/53
func GetLogger(config *config.GlobalConfig) *slog.Logger {
	return config.Log.Logger
}

func newLogger(config *config.GlobalConfig) *slog.Logger {
	handlerOptions := &slog.HandlerOptions{
		AddSource:   true,
		Level:       config.Log.Level,
		ReplaceAttr: nil,
	}

	var handler slog.Handler

	switch config.Log.Format {
	case "json":
		handler = slog.NewJSONHandler(config.Log.Writer, handlerOptions)
	case "text":
		fallthrough
	default:
		handler = slog.NewTextHandler(config.Log.Writer, handlerOptions)
	}

	return slog.New(handler)
}
