// Package view provides the view command for opampctl.
package view

import (
	"fmt"
	"slices"

	"github.com/spf13/cobra"

	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

// CommandOptions contains the options for the view command.
type CommandOptions struct {
	*config.GlobalConfig

	showSecrets bool
}

// NewCommand creates a new view command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
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
	cmd.Flags().BoolVar(&options.showSecrets, "show-secrets", false, "print secret values instead of redacting them")

	return cmd
}

// Prepare prepares the command to run.
func (opt *CommandOptions) Prepare(_ *cobra.Command, _ []string) error {
	// No preparation needed for view command
	return nil
}

// Run runs the command.
func (opt *CommandOptions) Run(cmd *cobra.Command, _ []string) error {
	configData := *opt.GlobalConfig
	if !opt.showSecrets {
		configData.Users = slices.Clone(configData.Users)
		for index := range configData.Users {
			auth := &configData.Users[index].Auth
			if auth.Password != "" {
				auth.Password = "<redacted>"
			}

			if auth.BearerToken != "" {
				auth.BearerToken = "<redacted>"
			}
		}
	}

	err := formatter.FormatYAML(cmd.OutOrStdout(), configData)
	if err != nil {
		return fmt.Errorf("failed to format config data: %w", err)
	}

	return nil
}
