// Package user implements the 'opampctl delete user' command.
package user

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
	// ErrUserIDRequired is returned when the target user ID is not provided.
	ErrUserIDRequired = errors.New("user ID is required")
)

// CommandOptions contains the options for the 'opampctl delete user' command.
type CommandOptions struct {
	*config.GlobalConfig

	// internal
	client *client.Client
}

// NewCommand creates a new 'opampctl delete user' command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Delete a user",
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
		return ErrUserIDRequired
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
		userID string
		err    error
	}

	results := lo.Map(ids, func(userID string, _ int) deleteResult {
		return deleteResult{
			userID: userID,
			err:    o.client.UserService.DeleteUser(cmd.Context(), userID),
		}
	})

	successfullyDeleted := lo.FilterMap(results, func(result deleteResult, _ int) (string, bool) {
		return result.userID, result.err == nil
	})
	failedToDelete := lo.FilterMap(results, func(result deleteResult, _ int) (string, bool) {
		return result.userID, result.err != nil
	})

	cmd.Printf("Successfully deleted %d user(s): %s\n",
		len(successfullyDeleted), strings.Join(successfullyDeleted, ", "))

	if len(failedToDelete) > 0 {
		cmd.PrintErrf("Failed to delete %d user(s): %s\n", len(failedToDelete), strings.Join(failedToDelete, ", "))
	}

	return nil
}
