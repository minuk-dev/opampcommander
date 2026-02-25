// Package agentgroup provides the create agentgroup command for opampctl.
package agentgroup

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
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

	var agentConfig *v1.AgentConfig

	if opt.agentConfigFile != "" {
		data, err := os.ReadFile(filepath.Clean(opt.agentConfigFile))
		if err != nil {
			return fmt.Errorf("failed to read agent config file: %w", err)
		}

		//exhaustruct:ignore
		agentConfig = &v1.AgentConfig{
			AgentRemoteConfig: &v1.AgentGroupRemoteConfig{
				AgentRemoteConfigSpec: &v1.AgentRemoteConfigSpec{
					Value:       string(data),
					ContentType: "text/yaml",
				},
			},
		}
	}

	//exhaustruct:ignore
	createRequest := &v1.AgentGroup{
		Metadata: v1.Metadata{
			Name:       opt.name,
			Attributes: opt.attributes,
			Priority:   opt.priority,
			Selector: v1.AgentSelector{
				IdentifyingAttributes:    opt.identifyingAttributesSelector,
				NonIdentifyingAttributes: opt.nonIdentifyingAttributeSelector,
			},
		},
		Spec: v1.Spec{
			AgentConfig: agentConfig,
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

//nolint:lll
type formattedAgentGroup struct {
	Name                             string            `json:"name"                             short:"name"                text:"name"                yaml:"name"`
	Attributes                       map[string]string `json:"attributes"                       short:"-"                   text:"-"                   yaml:"attributes"`
	IdentifyingAttributesSelector    map[string]string `json:"identifyingAttributesSelector"    short:"-"                   text:"-"                   yaml:"identifyingAttributesSelector"`
	NonIdentifyingAttributesSelector map[string]string `json:"nonIdentifyingAttributesSelector" short:"-"                   text:"-"                   yaml:"nonIdentifyingAttributesSelector"`
	DeletedAt                        *time.Time        `json:"deletedAt,omitempty"              short:"deletedAt,omitempty" text:"deletedAt,omitempty" yaml:"deletedAt,omitempty"`
}

func toFormattedAgentGroup(agentGroup *v1.AgentGroup) *formattedAgentGroup {
	return &formattedAgentGroup{
		Name:                             agentGroup.Metadata.Name,
		Attributes:                       agentGroup.Metadata.Attributes,
		IdentifyingAttributesSelector:    agentGroup.Metadata.Selector.IdentifyingAttributes,
		NonIdentifyingAttributesSelector: agentGroup.Metadata.Selector.NonIdentifyingAttributes,
		DeletedAt:                        switchToNilIfZero(agentGroup.Metadata.DeletedAt),
	}
}

func switchToNilIfZero(t *v1.Time) *time.Time {
	if t.IsZero() {
		return nil
	}

	return &t.Time
}
