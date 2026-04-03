// Package assign provides the rbac assign command for opampctl.
package assign

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
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

// CommandOptions contains the options for the rbac assign command.
type CommandOptions struct {
	*config.GlobalConfig

	// Flags
	userID string
	roleID string

	// internal state
	client *client.Client
}

// NewCommand creates a new rbac assign command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "assign",
		Short: "Assign a role to a user",
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

	cmd.Flags().StringVar(&options.userID, "user-id", "", "User ID to assign the role to (required)")
	cmd.Flags().StringVar(&options.roleID, "role-id", "", "Role ID to assign (required)")

	cmd.MarkFlagRequired("user-id") //nolint:errcheck,gosec
	cmd.MarkFlagRequired("role-id") //nolint:errcheck,gosec

	return cmd
}

// Prepare prepares the rbac assign command.
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

// Run executes the rbac assign command.
func (opt *CommandOptions) Run(cmd *cobra.Command, _ []string) error {
	req := &v1.AssignRoleRequest{
		UserID:     opt.userID,
		RoleID:     opt.roleID,
		AssignedBy: "", // Server-side: overridden with authenticated user's identity
	}

	err := opt.client.RBACService.AssignRole(cmd.Context(), req)
	if err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	cmd.Printf("Successfully assigned role %s to user %s\n", opt.roleID, opt.userID)

	return nil
}
