// Package remoteconfigschema implements the 'opampctl delete remoteconfigschema' command.
package remoteconfigschema

import (
	"errors"
	"fmt"
	"strings"

	"github.com/samber/lo"
	"github.com/spf13/cobra"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

// MinToComplete is the minimum number of characters to start completing.
const MinToComplete = 3

// ErrNameRequired is returned when the target schema name is not provided.
var ErrNameRequired = errors.New("remote config schema name is required")

// CommandOptions contains the options for the 'opampctl delete remoteconfigschema' command.
type CommandOptions struct {
	*config.GlobalConfig

	// flags
	namespace string

	// internal
	client *client.Client
}

// NewCommand creates a new 'opampctl delete remoteconfigschema' command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:               "remoteconfigschema",
		Short:             "delete remoteconfigschema",
		ValidArgsFunction: options.ValidArgsFunction,
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

	cmd.Flags().StringVarP(&options.namespace, "namespace", "n", "default", "Namespace")

	return cmd
}

// Prepare prepares the internal state before running the command.
func (o *CommandOptions) Prepare(_ *cobra.Command, args []string) error {
	if len(args) < 1 {
		return ErrNameRequired
	}

	client, err := clientutil.NewClient(o.GlobalConfig)
	if err != nil {
		return fmt.Errorf("failed to create authenticated client: %w", err)
	}

	o.client = client

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
			err: o.client.RemoteConfigSchemaService.DeleteRemoteConfigSchema(
				cmd.Context(), o.namespace, name,
			),
		}
	})

	successfullyDeleted := lo.FilterMap(results, func(r deleteResult, _ int) (string, bool) {
		return r.name, r.err == nil
	})
	failedToDelete := lo.FilterMap(results, func(r deleteResult, _ int) (string, bool) {
		return r.name, r.err != nil
	})

	cmd.Printf("Successfully deleted %d remote config schema(s): %s\n",
		len(successfullyDeleted), strings.Join(successfullyDeleted, ", "))

	if len(failedToDelete) > 0 {
		cmd.PrintErrf("Failed to delete %d remote config schema(s): %s\n",
			len(failedToDelete), strings.Join(failedToDelete, ", "))
	}

	return nil
}

// ValidArgsFunction provides dynamic completion for schema names.
func (o *CommandOptions) ValidArgsFunction(
	cmd *cobra.Command, _ []string, toComplete string,
) ([]string, cobra.ShellCompDirective) {
	client, err := clientutil.NewClient(o.GlobalConfig)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if len(toComplete) < MinToComplete {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	schemas, err := clientutil.ListRemoteConfigSchemaFully(cmd.Context(), client, o.namespace)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	matched := lo.Filter(schemas, func(s v1.RemoteConfigSchema, _ int) bool {
		return strings.Contains(strings.ToLower(s.Metadata.Name), strings.ToLower(toComplete))
	})

	names := lo.Map(matched, func(s v1.RemoteConfigSchema, _ int) string {
		return s.Metadata.Name
	})

	return names, cobra.ShellCompDirectiveNoFileComp
}
