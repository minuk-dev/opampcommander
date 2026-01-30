// Package deletecmd implements the 'opampctl delete' command.
// the package name is deletecmd to avoid conflict with the predeclared delete identifier.
package deletecmd

import (
	"github.com/spf13/cobra"

	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/deletecmd/agentgroup"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/deletecmd/agentpackage"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

// CommandOptions contains the options for the 'opampctl delete' command.
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
	cmd.AddCommand(agentpackage.NewCommand(agentpackage.CommandOptions{
		GlobalConfig: options.GlobalConfig,
	}))

	return cmd
}
