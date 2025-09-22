// Package agentgroup implements the 'opampctl get agentgroup' command.
package agentgroup

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/samber/lo"
	"github.com/spf13/cobra"

	v1agentgroup "github.com/minuk-dev/opampcommander/api/v1/agentgroup"
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

	displayedAgents := lo.Map(agentgroups, func(agentgroup v1agentgroup.AgentGroup, _ int) formattedAgentGroup {
		return toFormattedAgentGroup(agentgroup)
	})

	err = formatter.Format(cmd.OutOrStdout(), displayedAgents, formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format agentgroup: %w", err)
	}

	return nil
}

// Get retrieves the agent information for the given agent UIDs.
func (opt *CommandOptions) Get(cmd *cobra.Command, names []string) error {
	type AgentGroupWithErr struct {
		AgentGroup *v1agentgroup.AgentGroup
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
		return toFormattedAgentGroup(*a.AgentGroup)
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
	UID                              string            `json:"uid"                              short:"uid"       text:"uid"                 yaml:"uid"`
	Name                             string            `json:"name"                             short:"name"      text:"name"                yaml:"name"`
	Attributes                       map[string]string `json:"attributes"                       short:"-"         text:"-"                   yaml:"attributes"`
	IdentifyingAttributesSelector    map[string]string `json:"identifyingAttributesSelector"    short:"-"         text:"-"                   yaml:"identifyingAttributesSelector"`
	NonIdentifyingAttributesSelector map[string]string `json:"nonIdentifyingAttributesSelector" short:"-"         text:"-"                   yaml:"nonIdentifyingAttributesSelector"`
	CreatedAt                        time.Time         `json:"createdAt"                        short:"createdAt" text:"createdAt"           yaml:"createdAt"`
	CreatedBy                        string            `json:"createdBy"                        short:"createdBy" text:"createdBy"           yaml:"createdBy"`
	DeletedAt                        *time.Time        `json:"deletedAt,omitempty"              short:"-"         text:"deletedAt,omitempty" yaml:"deletedAt,omitempty"`
	DeletedBy                        *string           `json:"deletedBy,omitempty"              short:"-"         text:"deletedBy,omitempty" yaml:"deletedBy,omitempty"`
}

func toFormattedAgentGroup(agentGroup v1agentgroup.AgentGroup) formattedAgentGroup {
	return formattedAgentGroup{
		UID:                              agentGroup.UID.String(),
		Name:                             agentGroup.Name,
		Attributes:                       agentGroup.Attributes,
		IdentifyingAttributesSelector:    agentGroup.Selector.IdentifyingAttributes,
		NonIdentifyingAttributesSelector: agentGroup.Selector.NonIdentifyingAttributes,
		CreatedAt:                        agentGroup.CreatedAt,
		CreatedBy:                        agentGroup.CreatedBy,
		DeletedAt:                        agentGroup.DeletedAt,
		DeletedBy:                        agentGroup.DeletedBy,
	}
}
