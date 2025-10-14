// Package context provides the context command for opampctl.
package context

import (
	"github.com/spf13/cobra"

	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/context/ls"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/context/use"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

// CommandOptions contains the options for the context command.
type CommandOptions struct {
	*config.GlobalConfig
}

// NewCommand creates a new context command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Manage contexts",
	}

	cmd.AddCommand(use.NewCommand(use.CommandOptions{
		GlobalConfig: options.GlobalConfig,
	}))
	cmd.AddCommand(ls.NewCommand(ls.CommandOptions{
		GlobalConfig: options.GlobalConfig,
	}))

	return cmd
}
