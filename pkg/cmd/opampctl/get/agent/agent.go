// Package agent provides the command to get agent information.
package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/spf13/cobra"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
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

// CommandOptions contains the options for the agent command.
type CommandOptions struct {
	*config.GlobalConfig

	// flags
	formatType    string
	byAgentGroup  string
	namespace     string
	allNamespaces bool

	// internal
	client *client.Client
}

// NewCommand creates a new agent command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:               "agent",
		Short:             "agent",
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
	cmd.Flags().StringVarP(&options.formatType, "output", "o", "short", "Output format (short, text, json, yaml)")
	cmd.Flags().StringVar(&options.byAgentGroup, "agentgroup", "", "Filter agents by agent group name")
	cmd.Flags().StringVarP(&options.namespace, "namespace", "n", "default", "Namespace for the agent group filter")
	cmd.Flags().BoolVarP(&options.allNamespaces, "all-namespaces", "A", false, "List resources across all namespaces")

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

		return nil
	}

	agentUIDs := args

	err := opt.Get(cmd, agentUIDs)
	if err != nil {
		return fmt.Errorf("get failed: %w", err)
	}

	return nil
}

// ItemForCLI is a struct that represents an agent item for display.
type ItemForCLI struct {
	Namespace      string    `short:"Namespace"        text:"Namespace"`
	InstanceUID    uuid.UUID `short:"Instance UID"     text:"Instance UID"`
	ConnectionType string    `short:"Connection Type"  text:"Connection Type"`
	Connected      bool      `short:"Connected"        text:"Connected"`
	Healthy        bool      `short:"Healthy"          text:"Healthy"`
	SequenceNum    uint64    `short:"Sequence Num"     text:"Sequence Num"`
	StartedAt      string    `short:"Started At"       text:"Started At"`
	LastReportedAt string    `short:"Last Reported At" text:"Last Reported At"`
}

// List retrieves the list of agents.
func (opt *CommandOptions) List(cmd *cobra.Command) error {
	var (
		agents []v1.Agent
		err    error
	)

	if opt.allNamespaces {
		agents, err = opt.listAllNamespaces(cmd)
	} else {
		agents, err = opt.listSingleNamespace(cmd)
	}

	if err != nil {
		return err
	}

	return opt.formatAgents(cmd, agents)
}

// Get retrieves the agent information for the given agent UIDs.
func (opt *CommandOptions) Get(cmd *cobra.Command, ids []string) error {
	type AgentWithErr struct {
		Agent *v1.Agent
		Err   error
	}

	agentWithErrs := lo.Map(ids, func(id string, _ int) AgentWithErr {
		instanceUID, _ := uuid.Parse(id)
		agent, err := opt.client.AgentService.GetAgent(cmd.Context(), opt.namespace, instanceUID)

		return AgentWithErr{
			Agent: agent,
			Err:   err,
		}
	})

	agents := lo.Filter(agentWithErrs, func(a AgentWithErr, _ int) bool {
		return a.Err == nil
	})
	if len(agents) == 0 {
		cmd.Println("No agents found or all specified agents could not be retrieved.")

		return nil
	}

	displayedAgents := lo.Map(agents, func(a AgentWithErr, _ int) ItemForCLI {
		return toShortItemForCLI(*a.Agent)
	})

	err := formatter.Format(cmd.OutOrStdout(), displayedAgents, formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format agents: %w", err)
	}

	errs := lo.Filter(agentWithErrs, func(a AgentWithErr, _ int) bool {
		return a.Err != nil
	})
	if len(errs) > 0 {
		errMessages := lo.Map(errs, func(a AgentWithErr, _ int) string {
			return fmt.Sprintf("failed to get agent %s: %v", a.Agent.Metadata.InstanceUID, a.Err)
		})

		cmd.PrintErrf("Some agents could not be retrieved: %s", strings.Join(errMessages, ", "))
	}

	return nil
}

// ValidArgsFunction provides dynamic completion for agent instance UIDs.
func (opt *CommandOptions) ValidArgsFunction(
	cmd *cobra.Command, _ []string, toComplete string,
) ([]string, cobra.ShellCompDirective) {
	cli, err := clientutil.NewClient(opt.GlobalConfig)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	agentService := cli.AgentService

	// Use search API with the toComplete string as query
	resp, err := agentService.SearchAgents(
		cmd.Context(),
		opt.namespace,
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

func (opt *CommandOptions) listSingleNamespace(cmd *cobra.Command) ([]v1.Agent, error) {
	if opt.byAgentGroup == "" {
		agents, err := clientutil.ListAgentFully(cmd.Context(), opt.client, opt.namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to list agents: %w", err)
		}

		return agents, nil
	}

	agents, err := clientutil.ListAgentFullyByAgentGroup(
		cmd.Context(), opt.client, opt.namespace, opt.byAgentGroup,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents by agent group: %w", err)
	}

	return agents, nil
}

func (opt *CommandOptions) listAllNamespaces(cmd *cobra.Command) ([]v1.Agent, error) {
	agents, err := clientutil.ListAcrossNamespaces(
		cmd.Context(), opt.client,
		func(ctx context.Context, namespace string) ([]v1.Agent, error) {
			if opt.byAgentGroup == "" {
				return clientutil.ListAgentFully(ctx, opt.client, namespace)
			}

			return clientutil.ListAgentFullyByAgentGroup(ctx, opt.client, namespace, opt.byAgentGroup)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents across all namespaces: %w", err)
	}

	return agents, nil
}

func (opt *CommandOptions) formatAgents(cmd *cobra.Command, agents []v1.Agent) error {
	switch formatType := formatter.FormatType(opt.formatType); formatType {
	case formatter.SHORT, formatter.TEXT:
		displayedAgents := lo.Map(agents, func(agent v1.Agent, _ int) ItemForCLI {
			return toShortItemForCLI(agent)
		})

		err := formatter.Format(cmd.OutOrStdout(), displayedAgents, formatType)
		if err != nil {
			return fmt.Errorf("failed to format agents: %w", err)
		}
	case formatter.JSON, formatter.YAML:
		err := formatter.Format(cmd.OutOrStdout(), agents, formatType)
		if err != nil {
			return fmt.Errorf("failed to format agents: %w", err)
		}
	default:
		return fmt.Errorf("unsupported format type: %s, %w", opt.formatType, ErrCommandExecutionFailed)
	}

	return nil
}

func toShortItemForCLI(agent v1.Agent) ItemForCLI {
	var startedAt string
	if !agent.Status.ComponentHealth.StartTime.IsZero() {
		startedAt = agent.Status.ComponentHealth.StartTime.Format(time.DateTime)
	}

	return ItemForCLI{
		Namespace:      agent.Metadata.Namespace,
		InstanceUID:    agent.Metadata.InstanceUID,
		ConnectionType: agent.Status.ConnectionType,
		Connected:      agent.Status.Connected,
		Healthy:        agent.Status.ComponentHealth.Healthy,
		SequenceNum:    agent.Status.SequenceNum,
		StartedAt:      startedAt,
		LastReportedAt: agent.Status.LastReportedAt,
	}
}
