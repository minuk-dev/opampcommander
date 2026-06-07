// Package agent implements the 'opampctl delete agent' command.
package agent

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/spf13/cobra"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/deletecmd/internal/deleteutil"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

const (
	// MaxCompletionResults is the maximum number of completion results to return.
	MaxCompletionResults = 20
)

var (
	// ErrAgentInstanceUIDRequired is returned when the target agent instance UID is not provided.
	ErrAgentInstanceUIDRequired = errors.New("agent instance UID is required")
)

// CommandOptions contains the options for the 'opampctl delete agent' command.
type CommandOptions struct {
	*config.GlobalConfig

	// flags
	namespace string

	// internal
	client *client.Client
}

// NewCommand creates a new 'opampctl delete agent' command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:               "agent [instance-uid...]",
		Short:             "Delete disconnected agent(s)",
		Long:              "Delete one or more disconnected agents by instance UID. Connected agents cannot be deleted.",
		ValidArgsFunction: options.ValidArgsFunction,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := options.Prepare(cmd, args)
			if err != nil {
				return err
			}

			return options.Run(cmd, args)
		},
	}
	cmd.Flags().StringVarP(&options.namespace, "namespace", "n", "default", "Namespace of the agent")

	return cmd
}

// Prepare prepares the internal state before running the command.
func (o *CommandOptions) Prepare(_ *cobra.Command, args []string) error {
	if len(args) < 1 {
		return ErrAgentInstanceUIDRequired
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
	deleteutil.Run(cmd, "agent", ids, func(id string) error {
		instanceUID, parseErr := uuid.Parse(id)
		if parseErr != nil {
			return fmt.Errorf("invalid instance UID %q: %w", id, parseErr)
		}

		return o.client.AgentService.DeleteAgent(cmd.Context(), o.namespace, instanceUID)
	})

	return nil
}

// ValidArgsFunction provides dynamic completion for agent instance UIDs.
func (o *CommandOptions) ValidArgsFunction(
	cmd *cobra.Command, _ []string, toComplete string,
) ([]string, cobra.ShellCompDirective) {
	// unfortunately, we don't use o.client because this function is called without Prepare.
	cli, err := clientutil.NewClient(o.GlobalConfig)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	resp, err := cli.AgentService.SearchAgents(
		cmd.Context(),
		o.namespace,
		toComplete,
		client.WithLimit(MaxCompletionResults),
	)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	instanceUIDs := lo.Map(resp.Items, func(agent v1.Agent, _ int) string {
		return agent.Metadata.InstanceUID.String()
	})

	return instanceUIDs, cobra.ShellCompDirectiveNoFileComp
}
