// Package init provides the init command for opampctl.
package init

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

const (
	// DirectoryPermission defines the permission for directories created by the init command.
	DirectoryPermission = 0755 // Permission for directories
	// ConfigFilePermission defines the permission for configuration files created by the init command.
	ConfigFilePermission = 0600 // Permission for configuration files
)

var (
	// ErrUserCancelled is returned when the user cancels the operation.
	ErrUserCancelled = errors.New("user cancelled the operation")
)

// CommandOptions contains the options for the config command.
type CommandOptions struct {
	*config.GlobalConfig
}

// NewCommand creates a new config command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "init",
		Short: "init",
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			// To pass parents' PersistentPreRunE, we return nil here.
			return nil
		},
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

	return cmd
}

// Prepare prepares the command to run.
func (opt *CommandOptions) Prepare(_ *cobra.Command, _ []string) error {
	// No preparation needed for init command
	return nil
}

// Run runs the command.
func (opt *CommandOptions) Run(cmd *cobra.Command, _ []string) error {
	configFilename, err := opt.getConfigFilename(cmd)
	if err != nil {
		return fmt.Errorf("failed to get config filename: %w", err)
	}

	cmd.Printf("Using config file: %s\n", configFilename)

	filesystem := afero.NewOsFs()

	err = opt.ensureDirectoryExists(filesystem, configFilename)
	if err != nil {
		return fmt.Errorf("failed to ensure directory exists for config file: %w", err)
	}

	err = opt.handleExistingFile(cmd, filesystem, configFilename)
	if err != nil {
		return fmt.Errorf("failed to handle existing config file: %w", err)
	}

	err = opt.writeDefaultConfig(cmd, filesystem, configFilename)
	if err != nil {
		return fmt.Errorf("failed to write default config: %w", err)
	}

	return nil
}

func (opt *CommandOptions) getConfigFilename(cmd *cobra.Command) (string, error) {
	configFilename, err := cmd.Flags().GetString("config")
	if err != nil {
		return "", fmt.Errorf("failed to get config filename: %w", err)
	}

	if configFilename == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}

		configFilename = filepath.Join(homeDir, ".config", "opampcommander", "opampctl", "config.yaml")
	}

	return configFilename, nil
}

func (opt *CommandOptions) ensureDirectoryExists(filesystem afero.Fs, configFilename string) error {
	fileDir := filepath.Dir(configFilename)

	dirExists, err := afero.DirExists(filesystem, fileDir)
	if err != nil {
		return fmt.Errorf("failed to check if directory exists: %w", err)
	}

	if !dirExists {
		err = filesystem.MkdirAll(fileDir, DirectoryPermission)
		if err != nil {
			return fmt.Errorf("failed to create directory for config file: %w", err)
		}
	}

	return nil
}

func (opt *CommandOptions) handleExistingFile(cmd *cobra.Command, filesystem afero.Fs, configFilename string) error {
	fileExists, err := afero.Exists(filesystem, configFilename)
	if err != nil {
		return fmt.Errorf("failed to check if config file exists: %w", err)
	}

	if fileExists {
		cmd.Printf("Config file already exists: %s\n", configFilename)
		cmd.Printf("Do you want to overwrite it? (y/n): ")

		var response string

		_, err = fmt.Scanln(&response)
		if err != nil {
			return fmt.Errorf("failed to read user input: %w", err)
		}

		if strings.ToLower(response) != "y" {
			return ErrUserCancelled
		}
	}

	return nil
}

func (opt *CommandOptions) writeDefaultConfig(cmd *cobra.Command, filesystem afero.Fs, configFilename string) error {
	configFile, err := filesystem.OpenFile(configFilename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, ConfigFilePermission)
	if err != nil {
		return fmt.Errorf("failed to open config file for writing: %w", err)
	}

	defer func() {
		closeErr := configFile.Close()
		if closeErr != nil {
			cmd.PrintErrf("Failed to close config file: %v\n", closeErr)
		}
	}()

	encoder := yaml.NewEncoder(configFile)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}

	defaultConfig := config.NewDefaultGlobalConfig(homeDir)

	err = encoder.Encode(defaultConfig)
	if err != nil {
		return fmt.Errorf("failed to write default config to file: %w", err)
	}

	cmd.Printf("Config file initialized successfully: %s\n", configFilename)

	err = encoder.Close()
	if err != nil {
		return fmt.Errorf("failed to close YAML encoder: %w", err)
	}

	return nil
}
