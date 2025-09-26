// Package delete implements the 'opampctl delete' command.
package delete

import (
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/delete/agentgroup"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
	"github.com/spf13/cobra"
)

type CommandOptions struct {
	// GlobalConfig contains global configuration options.
	*config.GlobalConfig
}

// NewCommand creates a new 'opampctl delete' command.
func NewCommand(options CommandOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "delete",
	}

	cmd.AddCommand(agentgroup.NewCommand(agentgroup.CommandOptions{
		GlobalConfig: options.GlobalConfig,
	}))

	return cmd
}
