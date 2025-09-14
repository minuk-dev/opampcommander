// Package create provides the create command for opampctl.
package create

import (
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/create/agentgroup"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
	"github.com/spf13/cobra"
)

// CommandOptions contains the options for the create command.
type CommandOptions struct {
	*config.GlobalConfig
}

// NewCommand creates a new create command.
// It contains subcommands for creating resources.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "create",
		Short: "create",
	}

	cmd.AddCommand(agentgroup.NewCommand(agentgroup.CommandOptions{
		GlobalConfig: options.GlobalConfig,
	}))

	return cmd
}
