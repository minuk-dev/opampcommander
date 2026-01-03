// Package agent provides the command to set agent configurations.
package agent

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	v1agent "github.com/minuk-dev/opampcommander/api/v1/agent"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/cmdutil"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

var (
	// ErrTargetInstanceUIDNotSpecified is returned when the target agent instance UID is not specified.
	ErrTargetInstanceUIDNotSpecified = errors.New("target agent instance UID not specified")
	// ErrNewInstanceUIDNotSpecified is returned when the new instance UID is not specified.
	ErrNewInstanceUIDNotSpecified = errors.New("new instance UID not specified")
)

// CommandOptions contains the options for the set agent command.
type CommandOptions struct {
	*config.GlobalConfig

	// internal
	client *client.Client

	// flags
	formatType string

	targetInstanceUID uuid.UUID

	// new-instance-uid
	newInstanceUID       string
	parsedNewInstanceUID uuid.UUID // parsed after Prepare
}

// NewCommand creates a new set agent command.
func NewCommand(options CommandOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent [target-agent-instance-uid]",
		Short: "Set agent configurations",
		Long: `
  # Set a new instance UID for an agent
  opampctl set agent 550e8400-e29b-41d4-a716-446655440000 --new-instance-uid 550e8400-e29b-41d4-a716-446655440001

  # Set a new instance UID and output as JSON
  opampctl set agent 550e8400-e29b-41d4-a716-446655440000 --new-instance-uid \
  550e8400-e29b-41d4-a716-446655440001 -o json`,
		Args:              cobra.ExactArgs(1),
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
	cmd.Flags().StringVarP(&options.formatType, "output", "o", "yaml", "Output format (yaml|json|table)")
	cmd.Flags().StringVarP(&options.newInstanceUID, "new-instance-uid", "", "", "New instance UID to set for the agent")

	return cmd
}

// Prepare prepares the command options.
func (opts *CommandOptions) Prepare(_ *cobra.Command, args []string) error {
	// 0. Initialize client
	client, err := clientutil.NewClient(opts.GlobalConfig)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	opts.client = client

	// 1. Parse target agent instance UID
	if len(args) < 1 {
		return ErrTargetInstanceUIDNotSpecified
	}

	targetInstanceUID, err := uuid.Parse(args[0])
	if err != nil {
		return fmt.Errorf("invalid target agent instance UID: %w", err)
	}

	opts.targetInstanceUID = targetInstanceUID

	// 2. Parse new instance UID if set
	if opts.newInstanceUID == "" {
		return ErrNewInstanceUIDNotSpecified
	}

	parsedNewInstanceUID, err := uuid.Parse(opts.newInstanceUID)
	if err != nil {
		return fmt.Errorf("invalid new instance UID: %w", err)
	}

	opts.parsedNewInstanceUID = parsedNewInstanceUID

	return nil
}

// Run runs the command.
func (opts *CommandOptions) Run(cmd *cobra.Command, _ []string) error {
	request := v1agent.SetNewInstanceUIDRequest{
		NewInstanceUID: opts.parsedNewInstanceUID,
	}

	agent, err := opts.client.AgentService.SetAgentNewInstanceUID(cmd.Context(), opts.targetInstanceUID, request)
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

// ValidArgsFunction provides dynamic completion for agent instance UIDs.
// Only completes the first argument (agent instance UID).
func (opts *CommandOptions) ValidArgsFunction(
	cmd *cobra.Command, _ []string, toComplete string,
) ([]string, cobra.ShellCompDirective) {
	cli, err := clientutil.NewClient(opts.GlobalConfig)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	agentService := cli.AgentService

	instanceUids, err := cmdutil.AutoCompleteAgentInstanceUIDs(
		cmd.Context(),
		agentService,
		toComplete,
	)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	return instanceUids, cobra.ShellCompDirectiveNoFileComp
}
