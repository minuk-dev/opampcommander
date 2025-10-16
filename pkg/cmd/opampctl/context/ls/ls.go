// Package ls provides the ls command for opampctl context.
package ls

import (
	"github.com/spf13/cobra"

	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

// CommandOptions contains the options for the ls command.
type CommandOptions struct {
	*config.GlobalConfig
}

// NewCommand creates a new ls command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	return &cobra.Command{
		Use:   "ls",
		Short: "List all contexts",
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
	// No preparation needed for ls command
	return nil
}

// Run runs the command.
func (opt *CommandOptions) Run(cmd *cobra.Command, _ []string) error {
	if len(opt.Contexts) == 0 {
		cmd.Println("No contexts found")

		return nil
	}

	// Print contexts with current context marked
	for _, ctx := range opt.Contexts {
		if ctx.Name == opt.CurrentContext {
			cmd.Printf("* %s (current)\n", ctx.Name)
		} else {
			cmd.Printf("  %s\n", ctx.Name)
		}
	}

	return nil
}
