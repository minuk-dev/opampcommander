// Package agentgroup implements the 'opampctl get agentgroup' command.
package agentgroup

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/samber/lo"
	"github.com/spf13/cobra"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
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
	formatType string

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

// List retrieves the list of agents.
func (opt *CommandOptions) List(cmd *cobra.Command) error {
	agentgroups, err := clientutil.ListAgentGroupFully(cmd.Context(), opt.client)
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

// Get retrieves the agent information for the given agent UIDs.
func (opt *CommandOptions) Get(cmd *cobra.Command, names []string) error {
	type AgentGroupWithErr struct {
		AgentGroup *v1.AgentGroup
		Err        error
	}

	agentGroupsWithErr := lo.Map(names, func(name string, _ int) AgentGroupWithErr {
		agent, err := opt.client.AgentGroupService.GetAgentGroup(cmd.Context(), name)

		return AgentGroupWithErr{
			AgentGroup: agent,
			Err:        err,
		}
	})

	agentGroups := lo.Filter(agentGroupsWithErr, func(a AgentGroupWithErr, _ int) bool {
		return a.Err == nil
	})
	if len(agentGroupsWithErr) == 0 {
		cmd.Println("No agents found or all specified agents could not be retrieved.")

		return nil
	}

	displayedAgentGroups := lo.Map(agentGroups, func(a AgentGroupWithErr, _ int) formattedAgentGroup {
		return opt.toFormattedAgentGroup(*a.AgentGroup)
	})

	err := formatter.Format(cmd.OutOrStdout(), displayedAgentGroups, formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format agents: %w", err)
	}

	errs := lo.Filter(agentGroupsWithErr, func(a AgentGroupWithErr, _ int) bool {
		return a.Err != nil
	})
	if len(errs) > 0 {
		errMessages := lo.Map(errs, func(a AgentGroupWithErr, _ int) string {
			return a.Err.Error()
		})

		cmd.PrintErrf("Some agents could not be retrieved: %s", strings.Join(errMessages, ", "))
	}

	return nil
}

//nolint:lll
type formattedAgentGroup struct {
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
		}
	}

	return createdAt, createdBy, deletedAt, deletedBy
}

func (opt *CommandOptions) toFormattedAgentGroup(
	agentGroup v1.AgentGroup,
) formattedAgentGroup {
	// Extract timestamps and users from conditions
	createdAt, createdBy, deletedAt, deletedBy := extractConditionInfo(agentGroup.Status.Conditions)

	return formattedAgentGroup{
		Name:                             agentGroup.Metadata.Name,
		NumTotalAgents:                   agentGroup.Status.NumAgents,
		NumConnectedHealthyAgents:        agentGroup.Status.NumHealthyAgents,
		NumConnectedUnhealthyAgents:      agentGroup.Status.NumUnhealthyAgents,
		NumNotConnectedAgents:            agentGroup.Status.NumNotConnectedAgents,
		Attributes:                       agentGroup.Metadata.Attributes,
		IdentifyingAttributesSelector:    agentGroup.Metadata.Selector.IdentifyingAttributes,
		NonIdentifyingAttributesSelector: agentGroup.Metadata.Selector.NonIdentifyingAttributes,
		CreatedAt:                        createdAt,
		CreatedBy:                        createdBy,
		DeletedAt:                        deletedAt,
		DeletedBy:                        deletedBy,
	}
}
