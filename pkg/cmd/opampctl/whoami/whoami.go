// Package whoami provides the whoami command for opampctl.
package whoami

import (
	"fmt"

	"github.com/spf13/cobra"

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

	data := shortItemForCLI{
		Name:          currentUser.Name,
		AuthType:      currentUser.Auth.Type,
		Email:         switchIfNil(info.Email, "N/A"),
		Authenticated: info.Authenticated,
	}

	err = formatter.FormatText(cmd.OutOrStdout(), data)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return nil
}

type shortItemForCLI struct {
	Name          string `json:"name"          short:"NAME"      text:"NAME"          yaml:"name"`
	AuthType      string `json:"authType"      short:"AUTH_TYPE" text:"AUTH_TYPE"     yaml:"authType"`
	Email         string `json:"email"         short:"EMAIL"     text:"EMAIL"         yaml:"email"`
	Authenticated bool   `json:"authenticated" short:"AUTH"      text:"AUTHENTICATED" yaml:"authenticated"`
}

func switchIfNil[T any](value *T, defaultValue T) T {
	if value == nil {
		return defaultValue
	}

	return *value
}
