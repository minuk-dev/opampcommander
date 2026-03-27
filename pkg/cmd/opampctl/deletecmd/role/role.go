// Package role implements the 'opampctl delete role' command.
package role

import (
	"errors"
	"fmt"
	"strings"

	"github.com/samber/lo"
	"github.com/spf13/cobra"

	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

var (
	// ErrRoleIDRequired is returned when the target role ID is not provided.
	ErrRoleIDRequired = errors.New("role ID is required")
)

// CommandOptions contains the options for the 'opampctl delete role' command.
type CommandOptions struct {
	*config.GlobalConfig

	// internal
	client *client.Client
}

// NewCommand creates a new 'opampctl delete role' command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "role",
		Short: "Delete a role",
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

	return cmd
}

// Prepare prepares the internal state before running the command.
func (o *CommandOptions) Prepare(_ *cobra.Command, args []string) error {
	if len(args) < 1 {
		return ErrRoleIDRequired
	}

	cli, err := clientutil.NewClient(o.GlobalConfig)
	if err != nil {
		return fmt.Errorf("failed to create authenticated client: %w", err)
	}

	o.client = cli

	return nil
}

// Run runs the command.
func (o *CommandOptions) Run(cmd *cobra.Command, ids []string) error {
	type deleteResult struct {
		roleID string
		err    error
	}

	results := lo.Map(ids, func(roleID string, _ int) deleteResult {
		return deleteResult{
			roleID: roleID,
			err:    o.client.RoleService.DeleteRole(cmd.Context(), roleID),
		}
	})

	successfullyDeleted := lo.FilterMap(results, func(result deleteResult, _ int) (string, bool) {
		return result.roleID, result.err == nil
	})
	failedToDelete := lo.FilterMap(results, func(result deleteResult, _ int) (string, bool) {
		return result.roleID, result.err != nil
	})

	cmd.Printf("Successfully deleted %d role(s): %s\n",
		len(successfullyDeleted), strings.Join(successfullyDeleted, ", "))

	if len(failedToDelete) > 0 {
		cmd.PrintErrf("Failed to delete %d role(s): %s\n", len(failedToDelete), strings.Join(failedToDelete, ", "))
	}

	return nil
}
