// Package agentgroup implements the 'opampctl get agentgroup' command.
package agentgroup

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	pagination     listutil.Flags
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

// List retrieves one page, or streams all pages when requested.
func (opt *CommandOptions) List(cmd *cobra.Command) error {
	selectorOpts, err := opt.selectors.ListOptions()
	if err != nil {
		return fmt.Errorf("failed to list agent groups: %w", err)
	}

	listOpts := append([]client.ListOption{client.WithIncludeDeleted(opt.includeDeleted)}, selectorOpts...)
	fetch := func(
		ctx context.Context, namespace string, pageOpts ...client.ListOption,
	) (*v1.ListResponse[v1.AgentGroup], error) {
		pageOpts = append(pageOpts, listOpts...)
		if opt.agent != "" {
			return opt.client.AgentGroupService.ListAgentGroupsByAgent(ctx, namespace, opt.agent, pageOpts...)
		}

		return opt.client.AgentGroupService.ListAgentGroups(ctx, namespace, pageOpts...)
	}

	err = listutil.Run(cmd, opt.client, opt.pagination, opt.namespace, opt.allNamespaces,
		formatter.FormatType(opt.formatType), fetch, func(groups []v1.AgentGroup) any {
			return lo.Map(groups, func(group v1.AgentGroup, _ int) formattedAgentGroup {
				return opt.toFormattedAgentGroup(group)
			})
		})
	if err != nil {
		return fmt.Errorf("list agentgroup: %w", err)
	}

	return nil
}

// ListByAgent uses the same bounded listing path for membership queries.
func (opt *CommandOptions) ListByAgent(cmd *cobra.Command) error {
	return opt.List(cmd)
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
