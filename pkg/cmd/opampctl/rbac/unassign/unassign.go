// Package unassign provides the rbac unassign command for opampctl.
package unassign

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

var (
	// ErrUserIDRequired is returned when the user ID flag is missing.
	ErrUserIDRequired = errors.New("--user-id is required")
	// ErrRoleIDRequired is returned when the role ID flag is missing.
	ErrRoleIDRequired = errors.New("--role-id is required")
)

// CommandOptions contains the options for the rbac unassign command.
type CommandOptions struct {
	*config.GlobalConfig

	// Flags
	userID string
	roleID string

	// internal state
	client *client.Client
}

// NewCommand creates a new rbac unassign command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "unassign",
		Short: "Unassign a role from a user",
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

	cmd.Flags().StringVar(&options.userID, "user-id", "", "User ID to unassign the role from (required)")
	cmd.Flags().StringVar(&options.roleID, "role-id", "", "Role ID to unassign (required)")

	cmd.MarkFlagRequired("user-id") //nolint:errcheck,gosec
	cmd.MarkFlagRequired("role-id") //nolint:errcheck,gosec

	return cmd
}

// Prepare prepares the rbac unassign command.
func (opt *CommandOptions) Prepare(*cobra.Command, []string) error {
	if opt.userID == "" {
		return ErrUserIDRequired
	}

	if opt.roleID == "" {
		return ErrRoleIDRequired
	}

	cli, err := clientutil.NewClient(opt.GlobalConfig)
	if err != nil {
		return fmt.Errorf("failed to create authenticated client: %w", err)
	}

	opt.client = cli

	return nil
}

// Run executes the rbac unassign command.
func (opt *CommandOptions) Run(cmd *cobra.Command, _ []string) error {
	err := opt.client.RBACService.UnassignRole(cmd.Context(), opt.userID, opt.roleID)
	if err != nil {
		return fmt.Errorf("failed to unassign role: %w", err)
	}

	cmd.Printf("Successfully unassigned role %s from user %s\n", opt.roleID, opt.userID)

	return nil
}
