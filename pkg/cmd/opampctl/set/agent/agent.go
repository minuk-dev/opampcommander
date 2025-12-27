// Package agent provides the command to set agent configurations.
package agent

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/spf13/cobra"

	v1agent "github.com/minuk-dev/opampcommander/api/v1/agent"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

const (
	// MaxCompletionResults is the maximum number of completion results to return.
	MaxCompletionResults = 20
)

var (
	// ErrCommandExecutionFailed is returned when the command execution fails.
	ErrCommandExecutionFailed = errors.New("command execution failed")
)

// CommandOptions contains the options for the set agent command.
type CommandOptions struct {
	*config.GlobalConfig

	// flags
	formatType     string
	newInstanceUID string

	// internal
	client *client.Client
}

// NewCommand creates a new set agent command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Set agent configurations",
		Long: `Set various configurations for agents.

Available subcommands:
  new-instance-uid    Set a new instance UID for an agent`,
	}

	// Add subcommands
	cmd.AddCommand(newNewInstanceUIDCommand(options))

	return cmd
}

// newNewInstanceUIDCommand creates the new-instance-uid subcommand.
func newNewInstanceUIDCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "new-instance-uid [AGENT_INSTANCE_UID] [NEW_INSTANCE_UID]",
		Short: "Set a new instance UID for an agent",
		Long: `Set a new instance UID for an agent.

The agent will be notified of the new instance UID when it next connects to the server.

Examples:
  # Set a new instance UID for an agent
  opampctl set agent new-instance-uid 550e8400-e29b-41d4-a716-446655440000 550e8400-e29b-41d4-a716-446655440001

  # Set a new instance UID and output as JSON
  opampctl set agent new-instance-uid 550e8400-e29b-41d4-a716-446655440000 \
    550e8400-e29b-41d4-a716-446655440001 -o json`,
		Args:              cobra.ExactArgs(2), //nolint:mnd // exactly 2 args are required
		ValidArgsFunction: options.ValidArgsFunction,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse agent instance UID
			instanceUID, err := uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("invalid agent instance UID: %w", err)
			}

			// Get new instance UID from args
			options.newInstanceUID = args[1]

			err = options.Prepare(cmd, args)
			if err != nil {
				return err
			}

			return options.setNewInstanceUID(cmd, instanceUID)
		},
	}

	cmd.Flags().StringVarP(&options.formatType, "output", "o", "yaml", "Output format (yaml|json|table)")

	return cmd
}

// Prepare prepares the command for execution.
func (opts *CommandOptions) Prepare(*cobra.Command, []string) error {
	client, err := clientutil.NewClient(opts.GlobalConfig)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	opts.client = client

	return nil
}

// ValidArgsFunction provides dynamic completion for agent instance UIDs.
// Only completes the first argument (agent instance UID).
func (opts *CommandOptions) ValidArgsFunction(
	cmd *cobra.Command, args []string, toComplete string,
) ([]string, cobra.ShellCompDirective) {
	// Only provide completion for the first argument (agent instance UID)
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	cli, err := clientutil.NewClient(opts.GlobalConfig)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	agentService := cli.AgentService

	// Use search API with the toComplete string as query
	resp, err := agentService.SearchAgents(
		cmd.Context(),
		toComplete,
		client.WithLimit(MaxCompletionResults),
	)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	instanceUIDs := lo.Map(resp.Items, func(agent v1agent.Agent, _ int) string {
		return agent.Metadata.InstanceUID.String()
	})

	return instanceUIDs, cobra.ShellCompDirectiveNoFileComp
}

// setNewInstanceUID sets a new instance UID for an agent.
func (opts *CommandOptions) setNewInstanceUID(cmd *cobra.Command, instanceUID uuid.UUID) error {
	newUID, err := uuid.Parse(opts.newInstanceUID)
	if err != nil {
		return fmt.Errorf("invalid new instance UID: %w", err)
	}

	request := v1agent.SetNewInstanceUIDRequest{
		NewInstanceUID: newUID,
	}

	agent, err := opts.client.AgentService.SetAgentNewInstanceUID(cmd.Context(), instanceUID, request)
	if err != nil {
		return fmt.Errorf("failed to set new instance UID: %w", err)
	}

	formatType := formatter.FormatType(opts.formatType)

	err = formatter.Format(cmd.OutOrStdout(), agent, formatType)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return nil
}
