package agent

import (
	"fmt"

	"github.com/google/uuid"
	v1agent "github.com/minuk-dev/opampcommander/api/v1/agent"
	"github.com/minuk-dev/opampcommander/internal/opampctl/config"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

type CommandOptions struct {
	*config.GlobalConfig

	// internal
	client *client.Client
}

func NewCommand(options CommandOptions) *cobra.Command {
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

func (opt *CommandOptions) Prepare(_ *cobra.Command, _ []string) error {
	opt.client = client.NewClient(opt.Endpoint)
	return nil
}

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

func (opt *CommandOptions) List(cmd *cobra.Command) error {
	agents, err := opt.client.ListAgents()
	if err != nil {
		return fmt.Errorf("failed to list agents: %w", err)
	}

	cmd.Println(agents)

	return nil
}

func (opt *CommandOptions) Get(cmd *cobra.Command, ids []string) error {
	var agents []*v1agent.Agent

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
