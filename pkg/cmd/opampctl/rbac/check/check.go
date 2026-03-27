// Package check provides the rbac check command for opampctl.
package check

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
	// ErrResourceRequired is returned when the resource flag is missing.
	ErrResourceRequired = errors.New("--resource is required")
	// ErrActionRequired is returned when the action flag is missing.
	ErrActionRequired = errors.New("--action is required")
)

// CommandOptions contains the options for the rbac check command.
type CommandOptions struct {
	*config.GlobalConfig

	// Flags
	userID   string
	resource string
	action   string

	// internal state
	client *client.Client
}

// NewCommand creates a new rbac check command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check if a user has a specific permission",
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

	cmd.Flags().StringVar(&options.userID, "user-id", "", "User ID to check (required)")
	cmd.Flags().StringVar(&options.resource, "resource", "", "Resource to check access for (required)")
	cmd.Flags().StringVar(&options.action, "action", "", "Action to check access for (required)")

	cmd.MarkFlagRequired("user-id")  //nolint:errcheck,gosec
	cmd.MarkFlagRequired("resource") //nolint:errcheck,gosec
	cmd.MarkFlagRequired("action")   //nolint:errcheck,gosec

	return cmd
}

// Prepare prepares the rbac check command.
func (opt *CommandOptions) Prepare(*cobra.Command, []string) error {
	if opt.userID == "" {
		return ErrUserIDRequired
	}

	if opt.resource == "" {
		return ErrResourceRequired
	}

	if opt.action == "" {
		return ErrActionRequired
	}

	cli, err := clientutil.NewClient(opt.GlobalConfig)
	if err != nil {
		return fmt.Errorf("failed to create authenticated client: %w", err)
	}

	opt.client = cli

	return nil
}

// Run executes the rbac check command.
func (opt *CommandOptions) Run(cmd *cobra.Command, _ []string) error {
	req := &v1.CheckPermissionRequest{
		UserID:   opt.userID,
		Resource: opt.resource,
		Action:   opt.action,
	}

	result, err := opt.client.RBACService.CheckPermission(cmd.Context(), req)
	if err != nil {
		return fmt.Errorf("failed to check permission: %w", err)
	}

	if result.Allowed {
		cmd.Printf("ALLOWED: User %s has permission %s:%s\n", opt.userID, opt.resource, opt.action)
	} else {
		cmd.Printf("DENIED: User %s does not have permission %s:%s\n", opt.userID, opt.resource, opt.action)
	}

	return nil
}
