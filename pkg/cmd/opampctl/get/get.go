// Package get provides the get command for opampctl.
package get

import (
	"github.com/spf13/cobra"

	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/get/agent"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/get/agentgroup"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/get/connection"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

// CommandOptions contains the options for the get command.
type CommandOptions struct {
	*config.GlobalConfig
}

// NewCommand creates a new get command.
// It contains subcommands for getting resources.
func NewCommand(options CommandOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "get",
	}

	cmd.AddCommand(agent.NewCommand(agent.CommandOptions{
		GlobalConfig: options.GlobalConfig,
	}))
	cmd.AddCommand(connection.NewCommand(connection.CommandOptions{
		GlobalConfig: options.GlobalConfig,
	}))
	cmd.AddCommand(agentgroup.NewCommand(agentgroup.CommandOptions{
		GlobalConfig: options.GlobalConfig,
	}))

	return cmd
}
