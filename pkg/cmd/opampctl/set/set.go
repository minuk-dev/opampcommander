// Package set provides the set command for opampctl.
package set

import (
	"github.com/spf13/cobra"

	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/set/agent"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

// NewCommand creates a new set command.
func NewCommand(globalConfig *config.GlobalConfig) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set configurations",
		Long:  `Set various configurations for agents and other resources.`,
	}

	// Add subcommands
	cmd.AddCommand(agent.NewCommand(agent.CommandOptions{
		GlobalConfig: globalConfig,
	}))

	return cmd
}
