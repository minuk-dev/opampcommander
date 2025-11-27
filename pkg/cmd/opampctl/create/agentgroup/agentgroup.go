// Package agentgroup provides the create agentgroup command for opampctl.
package agentgroup

import (
	"fmt"
	"os"
	"path/filepath"
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
	priority                        int
	identifyingAttributesSelector   map[string]string
	nonIdentifyingAttributeSelector map[string]string
	formatType                      string
	agentConfigFile                 string

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
	cmd.Flags().StringToStringVar(&options.identifyingAttributesSelector, "identifying-attributes-selector",
		nil, "Identifying attributes selector for the agent group (key=value)")
	cmd.Flags().StringToStringVar(&options.identifyingAttributesSelector, "is",
		nil, "same as --identifying-attributes-selector")
	cmd.Flags().IntVarP(&options.priority, "priority", "p", 0,
		"Priority of the agent group. Higher priority agent groups are applied first.")
	cmd.Flags().StringToStringVar(&options.nonIdentifyingAttributeSelector, "non-identifying-attributes-selector",
		nil, "NonIdentifying attributes selector for the agent group (key=value)")
	cmd.Flags().StringToStringVar(&options.nonIdentifyingAttributeSelector, "ns",
		nil, "same as --non-identifying-attributes-selector")
	cmd.Flags().StringVarP(&options.formatType, "output", "o", "text", "Output format (text, json, yaml)")
	cmd.Flags().StringVar(&options.agentConfigFile, "agent-config", "", "Path to agent config file")

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

	var agentConfig *agentgroupv1.AgentConfig

	if opt.agentConfigFile != "" {
		data, err := os.ReadFile(filepath.Clean(opt.agentConfigFile))
		if err != nil {
			return fmt.Errorf("failed to read agent config file: %w", err)
		}

		agentConfig = &agentgroupv1.AgentConfig{
			Value: string(data),
		}
	}

	createRequest := &agentgroupv1.CreateRequest{
		Name:       opt.name,
		Attributes: opt.attributes,
		Priority:   opt.priority,
		Selector: agentgroupv1.AgentSelector{
			IdentifyingAttributes:    opt.identifyingAttributesSelector,
			NonIdentifyingAttributes: opt.nonIdentifyingAttributeSelector,
		},
		AgentConfig: agentConfig,
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

//nolint:lll
type formattedAgentGroup struct {
	Name                             string            `json:"name"                             short:"name"                text:"name"                yaml:"name"`
	Attributes                       map[string]string `json:"attributes"                       short:"-"                   text:"-"                   yaml:"attributes"`
	IdentifyingAttributesSelector    map[string]string `json:"identifyingAttributesSelector"    short:"-"                   text:"-"                   yaml:"identifyingAttributesSelector"`
	NonIdentifyingAttributesSelector map[string]string `json:"nonIdentifyingAttributesSelector" short:"-"                   text:"-"                   yaml:"nonIdentifyingAttributesSelector"`
	CreatedAt                        time.Time         `json:"createdAt"                        short:"createdAt"           text:"createdAt"           yaml:"createdAt"`
	CreatedBy                        string            `json:"createdBy"                        short:"createdBy"           text:"createdBy"           yaml:"createdBy"`
	DeletedAt                        *time.Time        `json:"deletedAt,omitempty"              short:"deletedAt,omitempty" text:"deletedAt,omitempty" yaml:"deletedAt,omitempty"`
	DeletedBy                        *string           `json:"deletedBy,omitempty"              short:"deletedBy,omitempty" text:"deletedBy,omitempty" yaml:"deletedBy,omitempty"`
}

func toFormattedAgentGroup(agentGroup *agentgroupv1.AgentGroup) *formattedAgentGroup {
	// Extract timestamps and users from conditions
	var createdAt time.Time
	var createdBy string
	var deletedAt *time.Time
	var deletedBy *string

	for _, condition := range agentGroup.Status.Conditions {
		switch condition.Type {
		case agentgroupv1.ConditionTypeCreated:
			if condition.Status == agentgroupv1.ConditionStatusTrue {
				createdAt = condition.LastTransitionTime
				createdBy = condition.Reason
			}
		case agentgroupv1.ConditionTypeDeleted:
			if condition.Status == agentgroupv1.ConditionStatusTrue {
				deletedAt = &condition.LastTransitionTime
				deletedBy = &condition.Reason
			}
		}
	}

	return &formattedAgentGroup{
		Name:                             agentGroup.Metadata.Name,
		Attributes:                       agentGroup.Metadata.Attributes,
		IdentifyingAttributesSelector:    agentGroup.Metadata.Selector.IdentifyingAttributes,
		NonIdentifyingAttributesSelector: agentGroup.Metadata.Selector.NonIdentifyingAttributes,
		CreatedAt:                        createdAt,
		CreatedBy:                        createdBy,
		DeletedAt:                        deletedAt,
		DeletedBy:                        deletedBy,
	}
}
