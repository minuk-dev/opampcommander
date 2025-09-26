// Package agentgroup implements the 'opampctl delete agentgroup' command.
package agentgroup

import (
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
	"github.com/spf13/cobra"
)

type CommandOptions struct {
	*config.GlobalConfig

	// Flags
	name string
}

// NewCommand creates a new 'opampctl delete agentgroup' command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "agentgroup",
		Short: "delete agentgroup",
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

	cmd.Flags().StringVar(&options.name, "name", "", "Name of the agent group (required)")
	cmd.MarkFlagRequired("name")

	return cmd
}

func (o *CommandOptions) Prepare(_ *cobra.Command, _ []string) error {
	return nil
}

func (o *CommandOptions) Run(_ *cobra.Command, _ []string) error {
	return nil
}
