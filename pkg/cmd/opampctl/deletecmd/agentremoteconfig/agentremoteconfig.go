// Package agentremoteconfig implements the 'opampctl delete agentremoteconfig' command.
package agentremoteconfig

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
	// ErrAgentRemoteConfigNameRequired is returned when the target agentremoteconfig name is not provided.
	ErrAgentRemoteConfigNameRequired = errors.New("agentremoteconfig name is required")
)

// CommandOptions contains the options for the 'opampctl delete agentremoteconfig' command.
type CommandOptions struct {
	*config.GlobalConfig

	// internal
	client *client.Client
}

// NewCommand creates a new 'opampctl delete agentremoteconfig' command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:               "agentremoteconfig",
		Short:             "delete agentremoteconfig",
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
		return ErrAgentRemoteConfigNameRequired
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
			err:  o.client.AgentRemoteConfigService.DeleteAgentRemoteConfig(cmd.Context(), name),
		}
	})

	successfullyDeleted := lo.FilterMap(results, func(r deleteResult, _ int) (string, bool) {
		return r.name, r.err == nil
	})
	failedToDelete := lo.FilterMap(results, func(r deleteResult, _ int) (string, bool) {
		return r.name, r.err != nil
	})

	cmd.Printf("Successfully deleted %d agentremoteconfig(s): %s\n",
		len(successfullyDeleted), strings.Join(successfullyDeleted, ", "))

	if len(failedToDelete) > 0 {
		cmd.PrintErrf("Failed to delete %d agentremoteconfig(s): %s\n", len(failedToDelete), strings.Join(failedToDelete, ", "))
	}

	return nil
}

// ValidArgsFunction provides dynamic completion for agentremoteconfig names.
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

	agentRemoteConfigs, err := clientutil.ListAgentRemoteConfigFully(cmd.Context(), client)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	matched := lo.Filter(agentRemoteConfigs, func(arc v1.AgentRemoteConfig, _ int) bool {
		return strings.Contains(strings.ToLower(arc.Metadata.Name), strings.ToLower(toComplete))
	})

	names := lo.Map(matched, func(arc v1.AgentRemoteConfig, _ int) string {
		return arc.Metadata.Name
	})

	return names, cobra.ShellCompDirectiveNoFileComp
}
