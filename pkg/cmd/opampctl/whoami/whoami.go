// Package whoami provides the whoami command for opampctl.
package whoami

import (
	"fmt"

	"github.com/spf13/cobra"

	v1auth "github.com/minuk-dev/opampcommander/api/v1/auth"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/configutil"
)

// CommandOptions contains the options for the whoami command.
type CommandOptions struct {
	*config.GlobalConfig

	// flags
	formatType string

	// internal fields to run the command
	client *client.Client
}

// NewCommand creates a new whoami command.
func NewCommand(options CommandOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "whoami",
		Short: "Display the current user and context information with server validation",
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

	cmd.Flags().StringVarP(&options.formatType, "format", "f", "text", "Output format (text, json, yaml)")

	return cmd
}

// Prepare prepares the command options before running the command.
func (o *CommandOptions) Prepare(cmd *cobra.Command, _ []string) error {
	config, err := configutil.ApplyCmdFlags(o.GlobalConfig, cmd)
	if err != nil {
		return fmt.Errorf("failed to apply command flags: %w", err)
	}

	client, err := clientutil.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to create authenticated client: %w", err)
	}

	o.client = client

	return nil
}

// Run executes display of the current user and context information.
func (o *CommandOptions) Run(cmd *cobra.Command, _ []string) error {
	currentUser := configutil.GetCurrentUser(o.GlobalConfig)

	info, err := o.client.AuthService.GetInfo()
	if err != nil {
		return fmt.Errorf("failed to get info from server: %w", err)
	}

	err = formatter.FormatTEXT(cmd.OutOrStdout(), struct {
		User *config.User
		Info *v1auth.InfoResponse
	}{
		User: currentUser,
		Info: info,
	})
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return nil
}
