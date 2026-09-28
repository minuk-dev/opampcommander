// Package instanceuid sets an agent's new instance UID.
package instanceuid

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/set/internal/resource"
	"github.com/minuk-dev/opampcommander/pkg/cmdutil"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

// CommandOptions contains the options for the set instance-uid command.
type CommandOptions struct {
	*config.GlobalConfig

	// internal
	client *client.Client

	// flags
	formatType string
	namespace  string

	targetInstanceUID uuid.UUID
	request           client.SetNewInstanceUIDRequest
	outputFormat      formatter.FormatType
}

// NewCommand creates a new set instance-uid command.
func NewCommand(options CommandOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "instance-uid (agent/UID | agent UID) NEW_UID",
		Short: "Set an agent's new instance UID",
		Example: `  opampctl set instance-uid agent/550e8400-e29b-41d4-a716-446655440000 550e8400-e29b-41d4-a716-446655440001
  opampctl set instance-uid agent 550e8400-e29b-41d4-a716-446655440000 550e8400-e29b-41d4-a716-446655440001 -o json`,
		Args:              cobra.RangeArgs(resource.MinArgs, resource.MaxArgs),
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
	cmd.Flags().StringVarP(&options.namespace, "namespace", "n", "default", "Namespace of the agent")

	return cmd
}

// Prepare prepares the command options.
func (opts *CommandOptions) Prepare(_ *cobra.Command, args []string) error {
	target, newUID, err := resource.Parse(args, "agent")
	if err != nil {
		return fmt.Errorf("parse agent target: %w", err)
	}

	targetInstanceUID, err := uuid.Parse(target)
	if err != nil {
		return fmt.Errorf("invalid target agent instance UID: %w", err)
	}

	opts.targetInstanceUID = targetInstanceUID

	parsedNewInstanceUID, err := uuid.Parse(newUID)
	if err != nil {
		return fmt.Errorf("invalid new instance UID: %w", err)
	}

	cli, err := clientutil.NewClient(opts.GlobalConfig)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	opts.client = cli
	opts.request = client.SetNewInstanceUIDRequest{NewInstanceUID: parsedNewInstanceUID}
	opts.outputFormat = formatter.FormatType(opts.formatType)

	return nil
}

// Run runs the command.
func (opts *CommandOptions) Run(cmd *cobra.Command, _ []string) error {
	agent, err := opts.client.AgentService.SetAgentNewInstanceUID(
		cmd.Context(), opts.namespace, opts.targetInstanceUID, opts.request,
	)
	if err != nil {
		return fmt.Errorf("failed to set new instance UID: %w", err)
	}

	err = formatter.Format(cmd.OutOrStdout(), agent, opts.outputFormat)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return nil
}

// ValidArgsFunction provides dynamic completion for agent instance UIDs.
// Completes the target agent instance UID in either supported resource form.
func (opts *CommandOptions) ValidArgsFunction(
	cmd *cobra.Command, args []string, toComplete string,
) ([]string, cobra.ShellCompDirective) {
	resourcePrefix := ""

	switch {
	case len(args) == 0 && strings.HasPrefix(toComplete, "agent/"):
		resourcePrefix = "agent/"
		toComplete = strings.TrimPrefix(toComplete, resourcePrefix)
	case len(args) == 0 && strings.HasPrefix("agent/", toComplete): //nolint:gocritic // completing a partially typed kind
		return []string{"agent/"}, cobra.ShellCompDirectiveNoFileComp
	case len(args) == 1 && args[0] == "agent":
	case len(args) != 0:
		return nil, cobra.ShellCompDirectiveNoFileComp
	default:
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	cli, err := clientutil.NewClient(opts.GlobalConfig)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	agentService := cli.AgentService

	instanceUids, err := cmdutil.AutoCompleteAgentInstanceUIDs(
		cmd.Context(),
		agentService,
		opts.namespace,
		toComplete,
	)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	for i := range instanceUids {
		instanceUids[i] = resourcePrefix + instanceUids[i]
	}

	return instanceUids, cobra.ShellCompDirectiveNoFileComp
}
