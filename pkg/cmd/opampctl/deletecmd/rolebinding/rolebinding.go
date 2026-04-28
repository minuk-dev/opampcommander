// Package rolebinding implements the 'opampctl delete rolebinding' command.
package rolebinding

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
	// ErrRoleBindingNameRequired is returned when the rolebinding name is not provided.
	ErrRoleBindingNameRequired = errors.New("rolebinding name is required")
)

// CommandOptions contains the options for the 'opampctl delete rolebinding' command.
type CommandOptions struct {
	*config.GlobalConfig

	// flags
	namespace string

	// internal
	client *client.Client
}

// NewCommand creates a new 'opampctl delete rolebinding' command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "rolebinding",
		Short: "Delete a RoleBinding",
		Long: `Delete a RoleBinding by name within a namespace.

Examples:
  opampctl delete rolebinding agent-viewer-production --namespace production
  opampctl delete rolebinding my-binding --namespace staging`,
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
	cmd.Flags().StringVarP(&options.namespace, "namespace", "n", "default", "Namespace of the role binding")

	return cmd
}

// Prepare prepares the internal state before running the command.
func (o *CommandOptions) Prepare(_ *cobra.Command, args []string) error {
	if len(args) < 1 {
		return ErrRoleBindingNameRequired
	}

	cli, err := clientutil.NewClient(o.GlobalConfig)
	if err != nil {
		return fmt.Errorf("failed to create authenticated client: %w", err)
	}

	o.client = cli

	return nil
}

// Run runs the command.
func (o *CommandOptions) Run(cmd *cobra.Command, names []string) error {
	type deleteResult struct {
		name string
		err  error
	}

	results := lo.Map(names, func(name string, _ int) deleteResult {
		return deleteResult{
			name: name,
			err:  o.client.RoleBindingService.DeleteRoleBinding(cmd.Context(), o.namespace, name),
		}
	})

	successfullyDeleted := lo.FilterMap(results, func(r deleteResult, _ int) (string, bool) {
		return r.name, r.err == nil
	})
	failedToDelete := lo.FilterMap(results, func(r deleteResult, _ int) (string, bool) {
		return r.name, r.err != nil
	})

	cmd.Printf("Successfully deleted %d rolebinding(s): %s\n",
		len(successfullyDeleted), strings.Join(successfullyDeleted, ", "))

	if len(failedToDelete) > 0 {
		cmd.PrintErrf("Failed to delete %d rolebinding(s): %s\n",
			len(failedToDelete), strings.Join(failedToDelete, ", "))
	}

	return nil
}
