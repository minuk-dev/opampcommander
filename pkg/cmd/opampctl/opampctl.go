// Package opampctl implements the opampctl command line tool.
// It provides a command line interface for interacting with the opampcommander server.
package opampctl

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	configCmd "github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/config"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/context"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/create"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/deletecmd"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/get"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/set"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/version"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/whoami"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/configutil"
)

// CommandOption contains the options for the opampctl command.
type CommandOption struct {
	globalConfig *config.GlobalConfig

	// viper
	viper *viper.Viper
}

// NewCommand creates a new opampctl command.
func NewCommand(options CommandOption) *cobra.Command {
	if options.globalConfig == nil {
		//exhaustruct:ignore
		options.globalConfig = &config.GlobalConfig{}
	}
	//exhaustruct:ignore
	cmd := &cobra.Command{
		PersistentPreRunE: options.PersistentPrepare,
		Use:               "opampctl",
		Short:             "opampctl",
	}
	configutil.CreateGlobalConfigFlags(cmd.PersistentFlags())
	cmd.AddCommand(get.NewCommand(get.CommandOptions{GlobalConfig: options.globalConfig}))
	cmd.AddCommand(set.NewCommand(options.globalConfig))
	cmd.AddCommand(deletecmd.NewCommand(deletecmd.CommandOptions{GlobalConfig: options.globalConfig}))
	cmd.AddCommand(create.NewCommand(create.CommandOptions{GlobalConfig: options.globalConfig}))
	cmd.AddCommand(configCmd.NewCommand(configCmd.CommandOptions{GlobalConfig: options.globalConfig}))
	cmd.AddCommand(context.NewCommand(context.CommandOptions{GlobalConfig: options.globalConfig}))
	cmd.AddCommand(whoami.NewCommand(whoami.CommandOptions{GlobalConfig: options.globalConfig}))
	cmd.AddCommand(version.NewCommand(version.CommandOptions{GlobalConfig: options.globalConfig}))

	return cmd
}

// PersistentPrepare prepares the command by binding flags and reading the configuration file.
// opampctl commands need to be prepare before running for each subcommand because all subcommands
// depend on the global configuration.
func (opt *CommandOption) PersistentPrepare(cmd *cobra.Command, _ []string) error {
	opt.viper = viper.New()

	err := opt.viper.BindPFlags(cmd.PersistentFlags())
	if err != nil {
		return fmt.Errorf("failed to bind flags: %w", err)
	}

	globalConfig, err := configutil.ApplyCmdFlags(opt.globalConfig, cmd)
	if err != nil {
		return fmt.Errorf("failed to apply command flags: %w", err)
	}

	if globalConfig.ConfigFilename != "" {
		opt.viper.SetConfigFile(globalConfig.ConfigFilename)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}

		opt.viper.AddConfigPath(filepath.Join(home, ".config", "opampcommander", "opampctl"))
		opt.viper.SetConfigName("config")
		opt.viper.SetConfigType("yaml")
	}

	opt.viper.AutomaticEnv()

	err = opt.viper.ReadInConfig()
	if err != nil {
		cmd.PrintErrf("Error reading config file: %v\n", err)
		cmd.PrintErrf("Please run `opampctl config init` to create a default config file.\n")

		return fmt.Errorf("failed to read config file: %w", err)
	}

	err = opt.viper.Unmarshal(opt.globalConfig)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}
