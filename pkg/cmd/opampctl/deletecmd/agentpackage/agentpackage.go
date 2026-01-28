// Package agentpackage implements the 'opampctl delete agentpackage' command.
package agentpackage

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

const (
	// MinToComplete is the minimum number of characters to start completing.
	MinToComplete = 3
)

var (
	// ErrAgentPackageNameRequired is returned when the target agentpackage name is not provided.
	ErrAgentPackageNameRequired = errors.New("agentpackage name is required")
)

// CommandOptions contains the options for the 'opampctl delete agentpackage' command.
type CommandOptions struct {
	*config.GlobalConfig

	// internal
	client *client.Client
}

// NewCommand creates a new 'opampctl delete agentpackage' command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:               "agentpackage",
		Short:             "delete agentpackage",
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

	return cmd
}

// Prepare prepares the internal state before running the command.
func (o *CommandOptions) Prepare(_ *cobra.Command, args []string) error {
	if len(args) < 1 {
		return ErrAgentPackageNameRequired
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
			err:  o.client.AgentPackageService.DeleteAgentPackage(cmd.Context(), name),
		}
	})

	successfullyDeleted := lo.FilterMap(results, func(r deleteResult, _ int) (string, bool) {
		return r.name, r.err == nil
	})
	failedToDelete := lo.FilterMap(results, func(r deleteResult, _ int) (string, bool) {
		return r.name, r.err != nil
	})

	cmd.Printf("Successfully deleted %d agentpackage(s): %s\n",
		len(successfullyDeleted), strings.Join(successfullyDeleted, ", "))

	if len(failedToDelete) > 0 {
		cmd.PrintErrf("Failed to delete %d agentpackage(s): %s\n", len(failedToDelete), strings.Join(failedToDelete, ", "))
	}

	return nil
}

// ValidArgsFunction provides dynamic completion for agentpackage names.
func (o *CommandOptions) ValidArgsFunction(
	cmd *cobra.Command, _ []string, toComplete string,
) ([]string, cobra.ShellCompDirective) {
	// unfortunately, we don't use o.client because this function is called without Prepare.
	client, err := clientutil.NewClient(o.GlobalConfig)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	if len(toComplete) < MinToComplete {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	agentPackages, err := clientutil.ListAgentPackageFully(cmd.Context(), client)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	matched := lo.Filter(agentPackages, func(ap v1.AgentPackage, _ int) bool {
		return strings.Contains(strings.ToLower(ap.Metadata.Name), strings.ToLower(toComplete))
	})

	names := lo.Map(matched, func(ap v1.AgentPackage, _ int) string {
		return ap.Metadata.Name
	})

	return names, cobra.ShellCompDirectiveNoFileComp
}
