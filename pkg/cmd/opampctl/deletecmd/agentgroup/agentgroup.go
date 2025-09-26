// Package agentgroup implements the 'opampctl delete agentgroup' command.
package agentgroup

import (
	"errors"
	"fmt"
	"strings"

	"github.com/samber/lo"
	"github.com/spf13/cobra"

	v1agentgroup "github.com/minuk-dev/opampcommander/api/v1/agentgroup"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

const (
	// MinToComplete is the minimum number of characters to start completing.
	MinToComplete = 3
)

var (
	// ErrAgentGroupNameRequired is returned when the target agentgroup name is not provided.
	ErrAgentGroupNameRequired = errors.New("agentgroup name is required")
)

// CommandOptions contains the options for the 'opampctl delete agentgroup' command.
type CommandOptions struct {
	*config.GlobalConfig

	// internal
	client *client.Client
}

// NewCommand creates a new 'opampctl delete agentgroup' command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:               "agentgroup",
		Short:             "delete agentgroup",
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
		return ErrAgentGroupNameRequired
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
			err:  o.client.AgentGroupService.DeleteAgentGroup(cmd.Context(), name),
		}
	})

	successfullyDeleted := lo.FilterMap(results, func(r deleteResult, _ int) (string, bool) {
		return r.name, r.err == nil
	})
	failedToDelete := lo.FilterMap(results, func(r deleteResult, _ int) (string, bool) {
		return r.name, r.err != nil
	})

	cmd.Printf("Successfully deleted %s agentgroup(s):\n", strings.Join(successfullyDeleted, ", "))

	if len(failedToDelete) > 0 {
		cmd.PrintErrf("Failed to delete %s agentgroup(s):\n", strings.Join(failedToDelete, ", "))
	}

	return nil
}

// ValidArgsFunction provides dynamic completion for agentgroup names.
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

	agentGroups, err := clientutil.ListAgentGroupFully(cmd.Context(), client)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	matched := lo.Filter(agentGroups, func(ag v1agentgroup.AgentGroup, _ int) bool {
		return strings.Contains(strings.ToLower(ag.Name), strings.ToLower(toComplete))
	})

	names := lo.Map(matched, func(ag v1agentgroup.AgentGroup, _ int) string {
		return ag.Name
	})

	return names, cobra.ShellCompDirectiveNoFileComp
}
