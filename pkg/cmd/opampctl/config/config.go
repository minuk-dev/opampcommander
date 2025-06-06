// Package config provides the command line for the opampctl tool.
package config

import (
	"github.com/spf13/cobra"

	initCmd "github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/config/init"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/config/view"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

// CommandOptions contains the options for the config command.
type CommandOptions struct {
	*config.GlobalConfig
}

// NewCommand creates a new config command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "config",
		Short: "config",
	}

	cmd.AddCommand(view.NewCommand(view.CommandOptions{
		GlobalConfig: options.GlobalConfig,
	}))
	cmd.AddCommand(initCmd.NewCommand(initCmd.CommandOptions{
		GlobalConfig: options.GlobalConfig,
	}))

	return cmd
}
