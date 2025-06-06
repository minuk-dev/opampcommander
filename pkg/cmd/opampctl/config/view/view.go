// Package view provides the view command for opampctl.
package view

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

// CommandOptions contains the options for the view command.
type CommandOptions struct {
	*config.GlobalConfig
}

// NewCommand creates a new view command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	return &cobra.Command{
		Use:   "view",
		Short: "view",
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
	// No preparation needed for view command
	return nil
}

// Run runs the command.
func (opt *CommandOptions) Run(cmd *cobra.Command, _ []string) error {
	configData := opt.GlobalConfig

	err := formatter.FormatYAML(cmd.OutOrStdout(), configData)
	if err != nil {
		return fmt.Errorf("failed to format config data: %w", err)
	}

	return nil
}
