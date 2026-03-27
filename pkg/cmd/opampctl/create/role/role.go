// Package role provides the create role command for opampctl.
package role

import (
	"fmt"

	"github.com/spf13/cobra"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

// CommandOptions contains the options for the create role command.
type CommandOptions struct {
	*config.GlobalConfig

	// Flags
	displayName string
	description string
	permissions []string
	formatType  string

	// internal state
	client *client.Client
}

// NewCommand creates a new create role command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "role",
		Short: "Create a new role",
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

	cmd.Flags().StringVar(&options.displayName, "display-name", "", "Display name of the role (required)")
	cmd.Flags().StringVar(&options.description, "description", "", "Description of the role")
	cmd.Flags().StringSliceVar(&options.permissions, "permissions", nil, "Permission IDs for the role")
	cmd.Flags().StringVarP(&options.formatType, "output", "o", "text", "Output format (text, json, yaml)")

	cmd.MarkFlagRequired("display-name") //nolint:errcheck,gosec

	return cmd
}

// Prepare prepares the create role command.
func (opt *CommandOptions) Prepare(*cobra.Command, []string) error {
	cli, err := clientutil.NewClient(opt.GlobalConfig)
	if err != nil {
		return fmt.Errorf("failed to create authenticated client: %w", err)
	}

	opt.client = cli

	return nil
}

// Run executes the create role command.
func (opt *CommandOptions) Run(cmd *cobra.Command, _ []string) error {
	//exhaustruct:ignore
	createRequest := &v1.Role{
		Kind:       v1.RoleKind,
		APIVersion: "v1",
		Spec: v1.RoleSpec{
			DisplayName: opt.displayName,
			Description: opt.description,
			Permissions: opt.permissions,
			IsBuiltIn:   false,
		},
	}

	role, err := opt.client.RoleService.CreateRole(cmd.Context(), createRequest)
	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	type formattedRole struct {
		UID         string `json:"uid"         short:"UID"          text:"UID"          yaml:"uid"`
		DisplayName string `json:"displayName" short:"DISPLAY_NAME" text:"DISPLAY_NAME" yaml:"displayName"`
		Description string `json:"description" short:"DESCRIPTION"  text:"DESCRIPTION"  yaml:"description"`
	}

	formatted := &formattedRole{
		UID:         role.Metadata.UID,
		DisplayName: role.Spec.DisplayName,
		Description: role.Spec.Description,
	}

	err = formatter.Format(cmd.OutOrStdout(), formatted, formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return nil
}
