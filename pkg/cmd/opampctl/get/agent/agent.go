// Package agent provides the command to get agent information.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/spf13/cobra"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/get/internal/getutil"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/get/internal/listutil"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/get/internal/selectorflags"
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
	formatType             string
	byAgentGroup           string
	namespace              string
	allNamespaces          bool
	decodeCapabilities     bool
	includeDisconnected    bool
	selectors              selectorflags.Flags
	pagination             listutil.Flags
	nonIdentifyingSelector map[string]string

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
	cmd.Flags().BoolVar(
		&options.decodeCapabilities, "decode-capabilities", false,
		"Decode the capabilities bitmask into human-readable flag names",
	)
	cmd.Flags().BoolVar(
		&options.includeDisconnected, "include-disconnected", false,
		"Include disconnected agents in the listing (by default only connected agents are shown)",
	)
	cmd.Flags().StringToStringVar(
		&options.nonIdentifyingSelector, "non-identifying-selector", nil,
		"Deprecated: use -l/--selector, which reaches non-identifying attributes too",
	)

	options.selectors.Register(cmd, selectorflags.Attributes)
	options.pagination.Register(cmd)

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
	ConnectionSync string    `short:"Connection Sync"  text:"Connection Sync"`
	Connected      bool      `short:"Connected"        text:"Connected"`
	Healthy        bool      `short:"Healthy"          text:"Healthy"`
	SequenceNum    uint64    `short:"Sequence Num"     text:"Sequence Num"`
	StartedAt      string    `short:"Started At"       text:"Started At"`
	LastReportedAt string    `short:"Last Reported At" text:"Last Reported At"`
}

// List retrieves the list of agents.
func (opt *CommandOptions) List(cmd *cobra.Command) error {
	opts, err := opt.listOptions()
	if err != nil {
		return err
	}

	var filter selectorflags.LocalFilter
	if opt.byAgentGroup != "" {
		filter, err = opt.selectors.LocalFilter()
		if err != nil {
			return fmt.Errorf("invalid agent filter: %w", err)
		}
	}

	fetch := func(
		ctx context.Context, namespace string, pageOpts ...client.ListOption,
	) (*v1.ListResponse[v1.Agent], error) {
		if opt.byAgentGroup == "" {
			return opt.client.AgentService.ListAgents(ctx, namespace, append(opts, pageOpts...)...)
		}

		resp, err := opt.client.AgentGroupService.ListAgentsByAgentGroup(ctx, namespace, opt.byAgentGroup,
			append(opt.connectionOptions(), pageOpts...)...)
		if err != nil {
			return nil, fmt.Errorf("list group members: %w", err)
		}

		resp.Items = lo.Filter(resp.Items, func(agent v1.Agent, _ int) bool {
			return filter.Matches(agent.Metadata.InstanceUID.String(), agent.Metadata.Description.IdentifyingAttributes,
				agent.Metadata.Description.NonIdentifyingAttributes) &&
				attributesMatch(agent.Metadata.Description.NonIdentifyingAttributes, opt.nonIdentifyingSelector)
		})

		return resp, nil
	}

	err = listutil.Run(cmd, opt.client, opt.pagination, opt.namespace, opt.allNamespaces,
		formatter.FormatType(opt.formatType), fetch, func(agents []v1.Agent) any {
			switch formatter.FormatType(opt.formatType) {
			case formatter.SHORT, formatter.TEXT:
				return lo.Map(agents, func(agent v1.Agent, _ int) ItemForCLI { return ToItemForCLI(agent) })
			case formatter.JSON, formatter.YAML:
				if opt.decodeCapabilities {
					return decodeAgentCapabilities(agents)
				}
			}

			return agents
		})
	if err != nil {
		return fmt.Errorf("list agent: %w", err)
	}

	return nil
}

// attributesMatch reports whether stored contains every key=value pair in selector.
func attributesMatch(stored, selector map[string]string) bool {
	for key, value := range selector {
		if stored[key] != value {
			return false
		}
	}

	return true
}

// Get retrieves the agent information for the given agent UIDs.
func (opt *CommandOptions) Get(cmd *cobra.Command, ids []string) error {
	items, lookupErr := getutil.Collect(cmd, "agent", ids, func(id string) (*v1.Agent, error) {
		instanceUID, err := uuid.Parse(id)
		if err != nil {
			return nil, fmt.Errorf("invalid UID: %w", err)
		}

		return opt.client.AgentService.GetAgent(cmd.Context(), opt.namespace, instanceUID)
	})

	displayed := lo.Map(items, func(item v1.Agent, _ int) ItemForCLI {
		return ToItemForCLI(item)
	})

	err := formatter.Format(cmd.OutOrStdout(), displayed, formatter.FormatType(opt.formatType))
	if err != nil {
		err = fmt.Errorf("failed to format agents: %w", err)
	}

	return errors.Join(lookupErr, err)
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

// listOptions renders every filtering flag the plain list endpoint understands,
// failing before the first request when an expression is malformed.
func (opt *CommandOptions) listOptions() ([]client.ListOption, error) {
	selectorOpts, err := opt.selectors.ListOptions()
	if err != nil {
		return nil, fmt.Errorf("failed to build the agent list options: %w", err)
	}

	opts := slices.Clone(selectorOpts)
	// --non-identifying-selector is deprecated but still forwarded.
	//nolint:staticcheck // deprecated flag, honoured until it is removed
	opts = append(opts, client.WithNonIdentifyingSelector(opt.nonIdentifyingSelector))

	return append(opts, opt.connectionOptions()...), nil
}

// connectionOptions builds the connection filter shared by every list path. By
// default it asks the server for connected agents only; --include-disconnected
// drops that filter, and so does an explicit status.connected field selector,
// which would otherwise be ANDed with it and silently return nothing.
func (opt *CommandOptions) connectionOptions() []client.ListOption {
	if opt.includeDisconnected || opt.selectors.ConstrainsField("status.connected") {
		return nil
	}

	return []client.ListOption{client.WithConnectedOnly(true)}
}

// decodeAgentCapabilities converts agents to a JSON-serializable form with capabilities
// expanded into human-readable flag names.
func decodeAgentCapabilities(agents []v1.Agent) []map[string]any {
	result := make([]map[string]any, 0, len(agents))

	for _, agent := range agents {
		jsonBytes, err := json.Marshal(agent)
		if err != nil {
			continue
		}

		var agentMap map[string]any

		err = json.Unmarshal(jsonBytes, &agentMap)
		if err != nil {
			continue
		}

		if metadata, ok := agentMap["metadata"].(map[string]any); ok {
			if _, ok := metadata["capabilities"].(float64); ok {
				metadata["capabilities"] = agent.Metadata.Capabilities.Names()
			}
		}

		result = append(result, agentMap)
	}

	return result
}

// ToItemForCLI converts an API agent to the shared CLI table representation.
func ToItemForCLI(agent v1.Agent) ItemForCLI {
	var startedAt string
	if !agent.Status.ComponentHealth.StartTime.IsZero() {
		startedAt = agent.Status.ComponentHealth.StartTime.Format(time.DateTime)
	}

	return ItemForCLI{
		Namespace:      agent.Metadata.Namespace,
		InstanceUID:    agent.Metadata.InstanceUID,
		ConnectionType: agent.Status.ConnectionType,
		ConnectionSync: agent.Status.ConnectionSettings.SyncStatus,
		Connected:      agent.Status.Connected,
		Healthy:        agent.Status.ComponentHealth.Healthy,
		SequenceNum:    agent.Status.SequenceNum,
		StartedAt:      startedAt,
		LastReportedAt: agent.Status.LastReportedAt,
	}
}
