// Package agentgroup implements the 'opampctl get agentgroup' command.
package agentgroup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/samber/lo"
	"github.com/spf13/cobra"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/get/internal/getutil"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/get/internal/selectorflags"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

var (
	// ErrCommandExecutionFailed is returned when the command execution fails.
	ErrCommandExecutionFailed = errors.New("command execution failed")
	// ErrAgentWithAllNamespaces is returned when --agent is combined with --all-namespaces.
	// An agent is looked up within a single namespace, so the two flags are mutually exclusive.
	ErrAgentWithAllNamespaces = errors.New("--agent cannot be combined with --all-namespaces")
)

// CommandOptions contains the options for the agent command.
type CommandOptions struct {
	*config.GlobalConfig

	// flags
	formatType     string
	includeDeleted bool
	namespace      string
	allNamespaces  bool
	selectors      selectorflags.Flags
	agent          string

	// internal
	client *client.Client
}

// NewCommand creates a new agent command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "agentgroup",
		Short: "agentgroup",
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
	cmd.Flags().BoolVar(&options.includeDeleted, "include-deleted", false, "Include soft-deleted agent groups")
	cmd.Flags().StringVarP(&options.namespace, "namespace", "n", "default", "Namespace of the agent group")
	cmd.Flags().BoolVarP(&options.allNamespaces, "all-namespaces", "A", false, "List resources across all namespaces")
	cmd.Flags().StringVar(&options.agent, "agent", "",
		"List only the agent groups that contain the agent with this instance UID")

	options.selectors.Register(cmd, selectorflags.Labels)

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
	if opt.agent != "" {
		if opt.allNamespaces {
			return ErrAgentWithAllNamespaces
		}

		err := opt.ListByAgent(cmd)
		if err != nil {
			return fmt.Errorf("list by agent failed: %w", err)
		}

		return nil
	}

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

// List retrieves the list of agents.
func (opt *CommandOptions) List(cmd *cobra.Command) error {
	selectorOpts, err := opt.selectors.ListOptions()
	if err != nil {
		return fmt.Errorf("failed to list agent groups: %w", err)
	}

	listOpts := append([]client.ListOption{client.WithIncludeDeleted(opt.includeDeleted)}, selectorOpts...)

	var agentgroups []v1.AgentGroup

	if opt.allNamespaces {
		agentgroups, err = opt.listAllNamespaces(cmd, listOpts...)
	} else {
		agentgroups, err = clientutil.ListAgentGroupFully(cmd.Context(), opt.client, opt.namespace, listOpts...)
	}

	if err != nil {
		return fmt.Errorf("failed to list agents: %w", err)
	}

	displayedAgents := make([]formattedAgentGroup, len(agentgroups))
	for idx, agentgroup := range agentgroups {
		displayedAgents[idx] = opt.toFormattedAgentGroup(agentgroup)
	}

	err = formatter.Format(cmd.OutOrStdout(), displayedAgents, formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format agentgroup: %w", err)
	}

	return nil
}

// ListByAgent streams matching groups a page at a time.
//
//nolint:funlen // Keep streaming output and cursor/error handling together.
func (opt *CommandOptions) ListByAgent(cmd *cobra.Command) error {
	selectorOpts, err := opt.selectors.ListOptions()
	if err != nil {
		return fmt.Errorf("list agent groups for agent %q: %w", opt.agent, err)
	}

	writer := cmd.OutOrStdout()

	jsonOutput := formatter.FormatType(opt.formatType) == formatter.JSON
	if jsonOutput {
		_, err = io.WriteString(writer, "[")
		if err != nil {
			return fmt.Errorf("write groups: %w", err)
		}
	}

	first := true

	token := ""
	for {
		opts := append([]client.ListOption{
			client.WithIncludeDeleted(opt.includeDeleted),
			client.WithLimit(clientutil.ChunkSize), client.WithContinueToken(token),
		}, selectorOpts...)

		resp, err := opt.client.AgentGroupService.ListAgentGroupsByAgent(cmd.Context(), opt.namespace, opt.agent, opts...)
		if err != nil {
			return fmt.Errorf("failed to list agent groups for agent %q in namespace %q: %w", opt.agent, opt.namespace, err)
		}

		displayed := lo.Map(resp.Items, func(group v1.AgentGroup, _ int) formattedAgentGroup {
			return opt.toFormattedAgentGroup(group)
		})
		if len(displayed) > 0 {
			err = opt.formatMembershipPage(writer, displayed, first)
			if err != nil {
				return err
			}

			first = false
		}

		if resp.Metadata.RemainingItemCount == 0 {
			break
		}

		if resp.Metadata.Continue == "" || resp.Metadata.Continue == token {
			return fmt.Errorf("%w: membership cursor did not advance", ErrCommandExecutionFailed)
		}

		token = resp.Metadata.Continue
	}

	if jsonOutput {
		_, err = io.WriteString(writer, "]\n")
		if err != nil {
			return fmt.Errorf("write groups: %w", err)
		}
	} else if first {
		err = opt.formatMembershipPage(writer, []formattedAgentGroup{}, true)
		if err != nil {
			return err
		}
	}

	return nil
}

// Get retrieves the agent information for the given agent UIDs.
func (opt *CommandOptions) Get(cmd *cobra.Command, ids []string) error {
	getOpts := []client.GetOption{client.WithGetIncludeDeleted(opt.includeDeleted)}

	items, lookupErr := getutil.Collect(cmd, "agentgroup", ids, func(id string) (*v1.AgentGroup, error) {
		return opt.client.AgentGroupService.GetAgentGroup(cmd.Context(), opt.namespace, id, getOpts...)
	})

	displayed := lo.Map(items, func(item v1.AgentGroup, _ int) formattedAgentGroup {
		return opt.toFormattedAgentGroup(item)
	})

	err := formatter.Format(cmd.OutOrStdout(), displayed, formatter.FormatType(opt.formatType))
	if err != nil {
		err = fmt.Errorf("failed to format agent groups: %w", err)
	}

	return errors.Join(lookupErr, err)
}

func (opt *CommandOptions) listAllNamespaces(
	cmd *cobra.Command, listOpts ...client.ListOption,
) ([]v1.AgentGroup, error) {
	agentgroups, err := clientutil.ListAcrossNamespaces(
		cmd.Context(), opt.client,
		func(ctx context.Context, namespace string) ([]v1.AgentGroup, error) {
			return clientutil.ListAgentGroupFully(ctx, opt.client, namespace, listOpts...)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list agent groups across all namespaces: %w", err)
	}

	return agentgroups, nil
}

//nolint:lll
type formattedAgentGroup struct {
	Namespace                        string            `json:"namespace"                        short:"namespace" text:"namespace"           yaml:"namespace"`
	Name                             string            `json:"name"                             short:"name"      text:"name"                yaml:"name"`
	NumTotalAgents                   int               `json:"numTotalAgents"                   short:"-"         text:"totalAgents"         yaml:"numTotalAgents"`
	NumConnectedHealthyAgents        int               `json:"numConnectedHealthyAgents"        short:"-"         text:"connected"           yaml:"numConnectedHealthyAgents"`
	NumConnectedUnhealthyAgents      int               `json:"numConnectedUnhealthyAgents"      short:"-"         text:"unhealthy"           yaml:"numConnectedUnhealthyAgents"`
	NumNotConnectedAgents            int               `json:"numNotConnectedAgents"            short:"-"         text:"disconnected"        yaml:"numNotConnectedAgents"`
	Attributes                       map[string]string `json:"attributes"                       short:"-"         text:"-"                   yaml:"attributes"`
	IdentifyingAttributesSelector    map[string]string `json:"identifyingAttributesSelector"    short:"-"         text:"-"                   yaml:"identifyingAttributesSelector"`
	NonIdentifyingAttributesSelector map[string]string `json:"nonIdentifyingAttributesSelector" short:"-"         text:"-"                   yaml:"nonIdentifyingAttributesSelector"`
	CreatedAt                        time.Time         `json:"createdAt"                        short:"createdAt" text:"createdAt"           yaml:"createdAt"`
	CreatedBy                        string            `json:"createdBy"                        short:"createdBy" text:"createdBy"           yaml:"createdBy"`
	DeletedAt                        *time.Time        `json:"deletedAt,omitempty"              short:"-"         text:"deletedAt,omitempty" yaml:"deletedAt,omitempty"`
	DeletedBy                        *string           `json:"deletedBy,omitempty"              short:"-"         text:"deletedBy,omitempty" yaml:"deletedBy,omitempty"`
}

func extractConditionInfo(conditions []v1.Condition) (time.Time, string, *time.Time, *string) {
	var (
		createdAt time.Time
		createdBy string
		deletedAt *time.Time
		deletedBy *string
	)

	for _, condition := range conditions {
		switch condition.Type {
		case v1.ConditionTypeCreated:
			if condition.Status == v1.ConditionStatusTrue {
				createdAt = condition.LastTransitionTime.Time
				createdBy = condition.Reason
			}
		case v1.ConditionTypeDeleted:
			if condition.Status == v1.ConditionStatusTrue {
				t := condition.LastTransitionTime.Time
				deletedAt = &t
				deletedBy = &condition.Reason
			}
		case v1.ConditionTypeUpdated,
			v1.ConditionTypeConnected,
			v1.ConditionTypeHealthy,
			v1.ConditionTypeConfigured,
			v1.ConditionTypeRegistered:
			// These condition types are not used for extracting info
		}
	}

	return createdAt, createdBy, deletedAt, deletedBy
}

func (opt *CommandOptions) toFormattedAgentGroup(
	agentGroup v1.AgentGroup,
) formattedAgentGroup {
	// Extract timestamps from metadata first, then fallback to conditions
	createdAt := agentGroup.Metadata.CreatedAt.Time

	// Get createdBy and deletedAt/deletedBy from conditions (createdBy is not in metadata)
	condCreatedAt, createdBy, deletedAt, deletedBy := extractConditionInfo(agentGroup.Status.Conditions)

	// Fallback to condition's createdAt if metadata doesn't have it
	if createdAt.IsZero() {
		createdAt = condCreatedAt
	}

	return formattedAgentGroup{
		Namespace:                        agentGroup.Metadata.Namespace,
		Name:                             agentGroup.Metadata.Name,
		NumTotalAgents:                   agentGroup.Status.NumAgents,
		NumConnectedHealthyAgents:        agentGroup.Status.NumHealthyAgents,
		NumConnectedUnhealthyAgents:      agentGroup.Status.NumUnhealthyAgents,
		NumNotConnectedAgents:            agentGroup.Status.NumNotConnectedAgents,
		Attributes:                       agentGroup.Metadata.Attributes,
		IdentifyingAttributesSelector:    agentGroup.Spec.Selector.IdentifyingAttributes,
		NonIdentifyingAttributesSelector: agentGroup.Spec.Selector.NonIdentifyingAttributes,
		CreatedAt:                        createdAt,
		CreatedBy:                        createdBy,
		DeletedAt:                        deletedAt,
		DeletedBy:                        deletedBy,
	}
}

func (opt *CommandOptions) formatMembershipPage(writer io.Writer, groups []formattedAgentGroup, first bool) error {
	if formatter.FormatType(opt.formatType) != formatter.JSON {
		err := formatter.Format(writer, groups, formatter.FormatType(opt.formatType))
		if err != nil {
			return fmt.Errorf("format agent groups: %w", err)
		}

		return nil
	}

	page, err := json.Marshal(groups)
	if err != nil {
		return fmt.Errorf("encode agent groups: %w", err)
	}

	separator := ""
	if !first {
		separator = ","
	}
	// The command owns the outer array; this page contributes only its elements.
	_, err = fmt.Fprintf(writer, "%s%s", separator, page[1:len(page)-1])
	if err != nil {
		return fmt.Errorf("write agent groups: %w", err)
	}

	return nil
}
