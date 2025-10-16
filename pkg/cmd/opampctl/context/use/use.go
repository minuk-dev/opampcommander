// Package use provides the use command for opampctl context.
package use

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

const (
	// configFilePermissions defines the file permissions for the config file.
	configFilePermissions = 0o600
)

var (
	// ErrContextNotFound is returned when the context is not found.
	ErrContextNotFound = errors.New("context does not exist")
)

// CommandOptions contains the options for the use command.
type CommandOptions struct {
	*config.GlobalConfig
}

// NewCommand creates a new use command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	return &cobra.Command{
		Use:   "use [context-name]",
		Short: "Switch to a different context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := options.Prepare(cmd, args)
			if err != nil {
				return err
			}

			err = options.Run(cmd, args)
			if err != nil {
				return err
			}

			return nil
		},
	}
}

// Prepare prepares the command to run.
func (opt *CommandOptions) Prepare(_ *cobra.Command, _ []string) error {
	// No preparation needed for use command
	return nil
}

// Run runs the command.
func (opt *CommandOptions) Run(cmd *cobra.Command, args []string) error {
	contextName := args[0]

	// Check if context exists
	contextExists := false

	for _, ctx := range opt.Contexts {
		if ctx.Name == contextName {
			contextExists = true

			break
		}
	}

	if !contextExists {
		return fmt.Errorf("%w: %q", ErrContextNotFound, contextName)
	}

	// Update current context
	opt.CurrentContext = contextName

	// Save to config file
	configPath := opt.ConfigFilename
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}

		configPath = filepath.Join(home, ".config", "opampcommander", "opampctl", "config.yaml")
	}

	// Read existing config file
	data, err := os.ReadFile(filepath.Clean(configPath))
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal to map to preserve order and comments
	var configMap map[string]any

	err = yaml.Unmarshal(data, &configMap)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Update currentContext
	configMap["currentContext"] = contextName

	// Marshal back to YAML
	updatedData, err := yaml.Marshal(configMap)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write back to file
	err = os.WriteFile(filepath.Clean(configPath), updatedData, configFilePermissions)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	cmd.Printf("Switched to context %q\n", contextName)

	return nil
}
