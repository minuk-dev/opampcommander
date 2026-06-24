// Package reconcile provides the reconcile command for opampctl. It re-runs, on demand, the
// side effects that normally fire when a resource is created or updated — useful for repairing
// drift on resources that predate a feature or whose triggers were missed.
package reconcile

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

// MaxCompletionResults is the maximum number of completion results to return.
const MaxCompletionResults = 20

// CommandOptions contains the options for the reconcile command.
type CommandOptions struct {
	GlobalConfig *config.GlobalConfig
}

// NewCommand creates a new reconcile command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "reconcile",
		Short: "reconcile resources",
		Long: "Re-run a resource's automatic side effects on demand (the same work performed when it is " +
			"created/updated or by the background reconcile loop).",
	}

	cmd.AddCommand(newReconcileRemoteConfigCommand(options))
	cmd.AddCommand(newReconcileAgentGroupCommand(options))
	cmd.AddCommand(newReconcileAgentCommand(options))

	return cmd
}

// newReconcileRemoteConfigCommand reconciles an agent remote config: it re-detects telemetry
// endpoints from the config's collector exporters and re-propagates it to referencing groups.
func newReconcileRemoteConfigCommand(options CommandOptions) *cobra.Command {
	var namespace string

	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:     "remoteconfig <name>",
		Aliases: []string{"agentremoteconfig"},
		Short:   "reconcile an agent remote config",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := clientutil.NewClient(options.GlobalConfig)
			if err != nil {
				return fmt.Errorf("failed to create API client: %w", err)
			}

			name := args[0]

			err = cli.AgentRemoteConfigService.ReconcileAgentRemoteConfig(context.Background(), namespace, name)
			if err != nil {
				return fmt.Errorf("failed to reconcile agent remote config: %w", err)
			}

			cmd.Printf("AgentRemoteConfig %s/%s reconciled successfully\n", namespace, name)

			return nil
		},
		ValidArgsFunction: remoteConfigNameCompletion(options, &namespace),
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Namespace of the agent remote config")

	return cmd
}

// newReconcileAgentGroupCommand reconciles an agent group: it re-applies the group's remote
// configs and connection settings to its matching agents.
func newReconcileAgentGroupCommand(options CommandOptions) *cobra.Command {
	var namespace string

	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "agentgroup <name>",
		Short: "reconcile an agent group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := clientutil.NewClient(options.GlobalConfig)
			if err != nil {
				return fmt.Errorf("failed to create API client: %w", err)
			}

			name := args[0]

			err = cli.AgentGroupService.ReconcileAgentGroup(context.Background(), namespace, name)
			if err != nil {
				return fmt.Errorf("failed to reconcile agent group: %w", err)
			}

			cmd.Printf("AgentGroup %s/%s reconciled successfully\n", namespace, name)

			return nil
		},
		ValidArgsFunction: agentGroupNameCompletion(options, &namespace),
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Namespace of the agent group")

	return cmd
}

// newReconcileAgentCommand reconciles an agent: it re-applies the agent groups that match the
// agent so it picks up its assigned remote configs and connection settings.
func newReconcileAgentCommand(options CommandOptions) *cobra.Command {
	var namespace string

	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "agent <id>",
		Short: "reconcile an agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := clientutil.NewClient(options.GlobalConfig)
			if err != nil {
				return fmt.Errorf("failed to create API client: %w", err)
			}

			agentUUID, err := uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("invalid agent ID format: %w", err)
			}

			err = cli.AgentService.ReconcileAgent(context.Background(), namespace, agentUUID)
			if err != nil {
				return fmt.Errorf("failed to reconcile agent: %w", err)
			}

			cmd.Printf("Agent %s/%s reconciled successfully\n", namespace, agentUUID)

			return nil
		},
		ValidArgsFunction: agentIDCompletion(options, &namespace),
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Namespace of the agent")

	return cmd
}

func remoteConfigNameCompletion(
	options CommandOptions,
	namespace *string,
) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		cli, err := clientutil.NewClient(options.GlobalConfig)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		items, err := clientutil.ListAgentRemoteConfigFully(cmd.Context(), cli, *namespace)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		names := lo.Map(items, func(item v1.AgentRemoteConfig, _ int) string {
			return item.Metadata.Name
		})

		return names, cobra.ShellCompDirectiveNoFileComp
	}
}

func agentGroupNameCompletion(
	options CommandOptions,
	namespace *string,
) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		cli, err := clientutil.NewClient(options.GlobalConfig)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		items, err := clientutil.ListAgentGroupFully(cmd.Context(), cli, *namespace)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		names := lo.Map(items, func(item v1.AgentGroup, _ int) string {
			return item.Metadata.Name
		})

		return names, cobra.ShellCompDirectiveNoFileComp
	}
}

func agentIDCompletion(
	options CommandOptions,
	namespace *string,
) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

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
