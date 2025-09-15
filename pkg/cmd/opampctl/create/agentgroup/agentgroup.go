// Package agentgroup provides the create agentgroup command for opampctl.
package agentgroup

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	agentgroupv1 "github.com/minuk-dev/opampcommander/api/v1/agentgroup"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

// CommandOptions contains the options for the create agentgroup command.
type CommandOptions struct {
	*config.GlobalConfig

	// Flags
	name                            string
	attributes                      map[string]string
	identifyingAttributesSelector   map[string]string
	nonIdentifyingAttributeSelector map[string]string
	formatType                      string

	// internal state
	client *client.Client
}

// NewCommand creates a new create agentgroup command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "agentgroup",
		Short: "create agentgroup",
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

	cmd.Flags().StringVar(&options.name, "name", "", "Name of the agent group (required)")
	cmd.Flags().StringToStringVar(&options.attributes, "attributes", nil, "Attributes of the agent group (key=value)")
	cmd.Flags().StringToStringVarP(&options.identifyingAttributesSelector,
		"is", "identifying-attributes-selector",
		nil, "Identifying attributes selector for the agent group (key=value)",
	)
	cmd.Flags().StringToStringVarP(&options.nonIdentifyingAttributeSelector,
		"ns", "non-identifying-attributes-selector",
		nil, "NonIdentifying attributes selector for the agent group (key=value)")
	cmd.Flags().StringVarP(&options.formatType, "output", "o", "text", "Output format (text, json, yaml)")

	cmd.MarkFlagRequired("name") //nolint:errcheck,gosec

	return cmd
}

// Prepare prepares the create agentgroup command.
func (opt *CommandOptions) Prepare(*cobra.Command, []string) error {
	client, err := clientutil.NewClient(opt.GlobalConfig)
	if err != nil {
		return fmt.Errorf("failed to create authenticated client: %w", err)
	}

	opt.client = client

	return nil
}

// Run executes the create agentgroup command.
func (opt *CommandOptions) Run(cmd *cobra.Command, _ []string) error {
	agentGroupService := opt.client.AgentGroupService

	createRequest := &agentgroupv1.CreateRequest{
		Name:       opt.name,
		Attributes: opt.attributes,
		Selector: agentgroupv1.AgentSelector{
			IdentifyingAttributes:    opt.identifyingAttributesSelector,
			NonIdentifyingAttributes: opt.nonIdentifyingAttributeSelector,
		},
	}

	agentGroup, err := agentGroupService.CreateAgentGroup(cmd.Context(), createRequest)
	if err != nil {
		return fmt.Errorf("failed to create agent group: %w", err)
	}

	err = formatter.Format(cmd.OutOrStdout(), toFormattedAgentGroup(agentGroup), formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return nil
}

type formattedAgentGroup struct {
	UID                              string            `text:"uid" short:"uid" json:"uid" yaml:"uid"`
	Name                             string            `text:"name" short:"name" json:"name" yaml:"name"`
	Attributes                       map[string]string `text:"-" short:"-" json:"attributes" yaml:"attributes"`
	IdentifyingAttributesSelector    map[string]string `text:"-" short:"-" json:"identifyingAttributesSelector" yaml:"identifyingAttributesSelector"`
	NonIdentifyingAttributesSelector map[string]string `text:"-" short:"-" json:"nonIdentifyingAttributesSelector" yaml:"nonIdentifyingAttributesSelector"`
	CreatedAt                        time.Time         `text:"createdAt" short:"createdAt" json:"createdAt" yaml:"createdAt"`
	CreatedBy                        string            `text:"createdBy" short:"createdBy" json:"createdBy" yaml:"createdBy"`
	DeletedAt                        *time.Time        `text:"deletedAt,omitempty" short:"deletedAt,omitempty" json:"deletedAt,omitempty" yaml:"deletedAt,omitempty"`
	DeletedBy                        *string           `text:"deletedBy,omitempty" short:"deletedBy,omitempty" json:"deletedBy,omitempty" yaml:"deletedBy,omitempty"`
}

func toFormattedAgentGroup(agentGroup *agentgroupv1.AgentGroup) *formattedAgentGroup {
	return &formattedAgentGroup{
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
