// Package configutil provides utilities for working with configuration files in the opampctl tool.
package configutil

import (
	"fmt"
	"io"
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
	flags.String("auth-flow", "",
		"Override the GitHub auth flow for this invocation: 'device' or 'browser'. "+
			"Empty falls back to the user's config (default: device).")
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
		Logger: newLogger(level, logFormat, cmd.ErrOrStderr()),
		Format: logFormat,
		Level:  level,
		Writer: cmd.ErrOrStderr(),
	}
	globalConfig.Output = cmd.OutOrStdout()

	authFlow, err := flags.GetString("auth-flow")
	if err != nil {
		return nil, fmt.Errorf("failed to get auth-flow flag: %w", err)
	}

	if authFlow != "" {
		applyAuthFlowOverride(globalConfig, authFlow)
	}

	return globalConfig, nil
}

// applyAuthFlowOverride sets the github auth flow on the current user, overriding any value
// from the config file for this invocation.
func applyAuthFlowOverride(globalConfig *config.GlobalConfig, flow string) {
	user := GetCurrentUser(globalConfig)
	if user == nil {
		return
	}

	user.Auth.Flow = flow
}

// GetLogger creates a new logger for the opampctl tool.
// It uses the default slog logger.
// In future, improve a logger: https://github.com/minuk-dev/opampcommander/issues/53
func GetLogger(config *config.GlobalConfig) *slog.Logger {
	return config.Log.Logger
}

func newLogger(level slog.Level, format string, writer io.Writer) *slog.Logger {
	handlerOptions := &slog.HandlerOptions{
		AddSource:   true,
		Level:       level,
		ReplaceAttr: nil,
	}

	var handler slog.Handler

	switch format {
	case "json":
		handler = slog.NewJSONHandler(writer, handlerOptions)
	case "text":
		fallthrough
	default:
		handler = slog.NewTextHandler(writer, handlerOptions)
	}

	return slog.New(handler)
}
