// Package agent provides the command to get agent information.
package agent

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/spf13/cobra"

	v1agent "github.com/minuk-dev/opampcommander/api/v1/agent"
	"github.com/minuk-dev/opampcommander/internal/opampctl/config"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
)

// CommandOptions contains the options for the agent command.
type CommandOptions struct {
	*config.GlobalConfig

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

	return cmd
}

// Prepare prepares the command to run.
func (opt *CommandOptions) Prepare(_ *cobra.Command, _ []string) error {
	opt.client = client.NewClient(opt.Endpoint)

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
	agents, err := opt.client.ListAgents()
	if err != nil {
		return fmt.Errorf("failed to list agents: %w", err)
	}

	err = formatter.FormatYAML(cmd.OutOrStdout(), agents)
	if err != nil {
		return fmt.Errorf("failed to format yaml: %w", err)
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
		agent, err := opt.client.GetAgent(instanceUID)
		if err != nil {
			return fmt.Errorf("failed to get agent: %w", err)
		}

		agents = append(agents, agent)
	}

	cmd.Println(agents)

	return nil
}
