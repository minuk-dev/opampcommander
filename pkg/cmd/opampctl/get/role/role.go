// Package role provides the command to get role information.
package role

import (
	"errors"
	"fmt"

	"github.com/samber/lo"
	"github.com/spf13/cobra"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

var (
	// ErrUnsupportedFormatType is returned when an unsupported format type is specified.
	ErrUnsupportedFormatType = errors.New("unsupported format type")
)

// CommandOptions contains the options for the role command.
type CommandOptions struct {
	*config.GlobalConfig

	// flags
	formatType string

	// internal
	client *client.Client
}

// NewCommand creates a new role command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "role",
		Short: "Get role information",
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
	cmd.Flags().StringVarP(&options.formatType, "output", "o", "short", "Output format (short, text, json, yaml)")

	return cmd
}

// Prepare prepares the command to run.
func (opt *CommandOptions) Prepare(*cobra.Command, []string) error {
	cli, err := clientutil.NewClient(opt.GlobalConfig)
	if err != nil {
		return fmt.Errorf("failed to create authenticated client: %w", err)
	}

	opt.client = cli

	return nil
}

// Run runs the command.
func (opt *CommandOptions) Run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return opt.List(cmd)
	}

	return opt.Get(cmd, args)
}

// ItemForCLI is a struct that represents a role item for display.
type ItemForCLI struct {
	UID         string `short:"UID"          text:"UID"`
	DisplayName string `short:"DISPLAY_NAME" text:"DISPLAY_NAME"`
	Description string `short:"DESCRIPTION"  text:"DESCRIPTION"`
	IsBuiltIn   bool   `short:"BUILT_IN"     text:"BUILT_IN"`
}

// List retrieves the list of roles.
func (opt *CommandOptions) List(cmd *cobra.Command) error {
	roles, err := clientutil.ListRoleFully(cmd.Context(), opt.client)
	if err != nil {
		return fmt.Errorf("failed to list roles: %w", err)
	}

	switch formatType := formatter.FormatType(opt.formatType); formatType {
	case formatter.SHORT, formatter.TEXT:
		displayedRoles := lo.Map(roles, func(role v1.Role, _ int) ItemForCLI {
			return toItemForCLI(role)
		})

		err = formatter.Format(cmd.OutOrStdout(), displayedRoles, formatType)
	case formatter.JSON, formatter.YAML:
		err = formatter.Format(cmd.OutOrStdout(), roles, formatType)
	default:
		return fmt.Errorf("unsupported format type: %s, %w", opt.formatType, ErrUnsupportedFormatType)
	}

	if err != nil {
		return fmt.Errorf("failed to format roles: %w", err)
	}

	return nil
}

// Get retrieves role information for the given role UIDs.
func (opt *CommandOptions) Get(cmd *cobra.Command, ids []string) error {
	type roleWithErr struct {
		Role *v1.Role
		Err  error
	}

	results := lo.Map(ids, func(id string, _ int) roleWithErr {
		role, err := opt.client.RoleService.GetRole(cmd.Context(), id)

		return roleWithErr{
			Role: role,
			Err:  err,
		}
	})

	roles := lo.Filter(results, func(result roleWithErr, _ int) bool {
		return result.Err == nil
	})
	if len(roles) == 0 {
		cmd.Println("No roles found or all specified roles could not be retrieved.")

		return nil
	}

	displayedRoles := lo.Map(roles, func(result roleWithErr, _ int) ItemForCLI {
		return toItemForCLI(*result.Role)
	})

	err := formatter.Format(cmd.OutOrStdout(), displayedRoles, formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format roles: %w", err)
	}

	return nil
}

func toItemForCLI(role v1.Role) ItemForCLI {
	return ItemForCLI{
		UID:         role.Metadata.UID,
		DisplayName: role.Spec.DisplayName,
		Description: role.Spec.Description,
		IsBuiltIn:   role.Spec.IsBuiltIn,
	}
}
