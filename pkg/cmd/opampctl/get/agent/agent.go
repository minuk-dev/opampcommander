// Package agent provides the command to get agent information.
package agent

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/spf13/cobra"

	v1agent "github.com/minuk-dev/opampcommander/api/v1/agent"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

var (
	// ErrCommandExecutionFailed is returned when the command execution fails.
	ErrCommandExecutionFailed = errors.New("command execution failed")
)

// CommandOptions contains the options for the agent command.
type CommandOptions struct {
	*config.GlobalConfig

	// flags
	formatType   string
	byAgentGroup string

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
	cmd.Flags().StringVarP(&options.formatType, "output", "o", "short", "Output format (short, text, json, yaml)")
	cmd.Flags().StringVar(&options.byAgentGroup, "agentgroup", "", "Filter agents by agent group name")

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
		agents []v1agent.Agent
		err    error
	)

	if opt.byAgentGroup == "" {
		agents, err = clientutil.ListAgentFully(cmd.Context(), opt.client)
		if err != nil {
			return fmt.Errorf("failed to list agents: %w", err)
		}
	} else {
		agents, err = clientutil.ListAgentFullyByAgentGroup(cmd.Context(), opt.client, opt.byAgentGroup)
		if err != nil {
			return fmt.Errorf("failed to list agents by agent group: %w", err)
		}
	}

	switch formatType := formatter.FormatType(opt.formatType); formatType {
	case formatter.SHORT, formatter.TEXT:
		displayedAgents := lo.Map(agents, func(agent v1agent.Agent, _ int) ItemForCLI {
			return toShortItemForCLI(agent)
		})
		err = formatter.Format(cmd.OutOrStdout(), displayedAgents, formatType)
	case formatter.JSON, formatter.YAML:
		err = formatter.Format(cmd.OutOrStdout(), agents, formatType)
	default:
		return fmt.Errorf("unsupported format type: %s, %w", opt.formatType, ErrCommandExecutionFailed)
	}

	if err != nil {
		return fmt.Errorf("failed to format agents: %w", err)
	}

	return nil
}

// Get retrieves the agent information for the given agent UIDs.
func (opt *CommandOptions) Get(cmd *cobra.Command, ids []string) error {
	type AgentWithErr struct {
		Agent *v1agent.Agent
		Err   error
	}

	agentWithErrs := lo.Map(ids, func(id string, _ int) AgentWithErr {
		instanceUID, _ := uuid.Parse(id)
		agent, err := opt.client.AgentService.GetAgent(cmd.Context(), instanceUID)

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

func toShortItemForCLI(agent v1agent.Agent) ItemForCLI {
	return ItemForCLI{
		InstanceUID:    agent.Metadata.InstanceUID,
		ConnectionType: agent.Status.ConnectionType,
		Connected:      agent.Status.Connected,
		Healthy:        agent.Status.ComponentHealth.Healthy,
		SequenceNum:    agent.Status.SequenceNum,
		StartedAt:      time.Unix(agent.Status.ComponentHealth.StartTimeUnix, 0).Format(time.DateTime),
		LastReportedAt: agent.Status.LastReportedAt,
	}
}
