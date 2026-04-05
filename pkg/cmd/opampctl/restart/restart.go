// Package restart provides the restart command for opampctl.
package restart

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/spf13/cobra"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

const (
	// MaxCompletionResults is the maximum number of completion results to return.
	MaxCompletionResults = 20
)

// CommandOptions contains the options for the restart command.
type CommandOptions struct {
	GlobalConfig *config.GlobalConfig
}

// NewCommand creates a new restart command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "restart",
		Short: "restart resources",
	}

	cmd.AddCommand(newRestartAgentCommand(options))

	return cmd
}

// newRestartAgentCommand creates a new restart agent command.
func newRestartAgentCommand(options CommandOptions) *cobra.Command {
	var (
		agentID   string
		namespace string
	)

	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "restart agent",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := context.Background()

			client, err := clientutil.NewClient(options.GlobalConfig)
			if err != nil {
				return fmt.Errorf("failed to create API client: %w", err)
			}

			agentUUID, err := uuid.Parse(agentID)
			if err != nil {
				return fmt.Errorf("invalid agent ID format: %w", err)
			}

			_, err = client.AgentService.RestartAgent(ctx, namespace, agentUUID)
			if err != nil {
				return fmt.Errorf("failed to restart agent: %w", err)
			}

			cmd.Printf("Agent %s restarted successfully\n", agentID)

			return nil
		},
	}

	cmd.Flags().StringVar(&agentID, "id", "", "agent ID to restart")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Namespace of the agent")
	_ = cmd.MarkFlagRequired("id")

	_ = cmd.RegisterFlagCompletionFunc("id",
		restartAgentIDCompletion(options, &namespace))

	return cmd
}

func restartAgentIDCompletion(
	options CommandOptions,
	namespace *string,
) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, _ []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		cli, err := clientutil.NewClient(options.GlobalConfig)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		resp, err := cli.AgentService.SearchAgents(
			cmd.Context(),
			*namespace,
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
}
