// Package agent provides the command to get agent information.
package agent

import (
	"context"
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

// CommandOptions contains the options for the agent command.
type CommandOptions struct {
	*config.GlobalConfig

	// flags
	formatType string

	// internal
	client *client.Client
}

// NewCommand creates a new agent command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "agent",
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
	cmd.Flags().StringVarP(&options.formatType, "format", "f", "short", "Output format (short, text, json, yaml)")

	return cmd
}

// Prepare prepares the command to run.
func (opt *CommandOptions) Prepare(*cobra.Command, []string) error {
	config := opt.GlobalConfig

	client, err := clientutil.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to create authenticated client: %w", err)
	}

	opt.client = client

	return nil
}

// Run runs the command.
func (opt *CommandOptions) Run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		err := opt.List(cmd)
		if err != nil {
			return fmt.Errorf("list failed: %w", err)
		}
	}

	agentUIDs := args

	err := opt.Get(cmd, agentUIDs)
	if err != nil {
		return fmt.Errorf("get failed: %w", err)
	}

	return nil
}

// List retrieves the list of agents.
func (opt *CommandOptions) List(cmd *cobra.Command) error {
	err := clientutil.ListAgentFully(cmd.Context(), opt.client, func(_ context.Context, agents []v1agent.Agent) error {
		if len(agents) == 0 {
			return nil
		}

		err := formatter.Format(cmd.OutOrStdout(), agents, formatter.FormatType(opt.formatType))
		if err != nil {
			return fmt.Errorf("failed to format agents: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to list agents: %w", err)
	}

	return nil
}

// Get retrieves the agent information for the given agent UIDs.
func (opt *CommandOptions) Get(cmd *cobra.Command, ids []string) error {
	agents := make([]*v1agent.Agent, 0, len(ids))
	instanceUIDs := lo.Map(ids, func(id string, _ int) uuid.UUID {
		instanceUID, _ := uuid.Parse(id)

		return instanceUID
	})

	for _, instanceUID := range instanceUIDs {
		agent, err := opt.client.AgentService.GetAgent(cmd.Context(), instanceUID)
		if err != nil {
			return fmt.Errorf("failed to get agent: %w", err)
		}

		agents = append(agents, agent)
	}

	cmd.Println(agents)

	return nil
}
