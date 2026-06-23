// Package agentgroup provides the create agentgroup command for opampctl.
package agentgroup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/create/internal/yamlfile"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

// ErrNameRequired is returned when --name is missing and --file is not used.
var ErrNameRequired = errors.New("--name is required (or use --file)")

// CommandOptions contains the options for the create agentgroup command.
type CommandOptions struct {
	*config.GlobalConfig

	// Flags
	name                            string
	namespace                       string
	attributes                      map[string]string
	priority                        int
	identifyingAttributesSelector   map[string]string
	nonIdentifyingAttributeSelector map[string]string
	formatType                      string
	agentConfigFile                 string
	file                            string

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
	cmd.Flags().StringVarP(&options.namespace, "namespace", "n", "default", "Namespace of the agent group")
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
	cmd.Flags().StringVarP(&options.file, "file", "f", "",
		"Path to a full AgentGroup YAML definition. When set, individual CLI flags are ignored.")

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
	createRequest, namespace, err := opt.buildRequest()
	if err != nil {
		return err
	}

	agentGroup, err := opt.client.AgentGroupService.CreateAgentGroup(cmd.Context(), namespace, createRequest)
	if err != nil {
		return fmt.Errorf("failed to create agent group: %w", err)
	}

	err = formatter.Format(cmd.OutOrStdout(), toFormattedAgentGroup(agentGroup), formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return nil
}

func (opt *CommandOptions) buildRequest() (*v1.AgentGroup, string, error) {
	if opt.file != "" {
		//exhaustruct:ignore
		req := &v1.AgentGroup{}

		err := yamlfile.Load(opt.file, req)
		if err != nil {
			return nil, "", fmt.Errorf("load agent group from %s: %w", opt.file, err)
		}

		namespace := req.Metadata.Namespace
		if namespace == "" {
			namespace = opt.namespace
		}

		return req, namespace, nil
	}

	if opt.name == "" {
		return nil, "", ErrNameRequired
	}

	agentConfig, err := opt.buildInlineAgentConfig()
	if err != nil {
		return nil, "", err
	}

	//exhaustruct:ignore
	return &v1.AgentGroup{
		Metadata: v1.Metadata{
			Name:       opt.name,
			Attributes: opt.attributes,
		},
		Spec: v1.Spec{
			Priority: opt.priority,
			Selector: v1.AgentSelector{
				IdentifyingAttributes:    opt.identifyingAttributesSelector,
				NonIdentifyingAttributes: opt.nonIdentifyingAttributeSelector,
			},
			AgentConfig: agentConfig,
		},
	}, opt.namespace, nil
}

func (opt *CommandOptions) buildInlineAgentConfig() (*v1.AgentConfig, error) {
	if opt.agentConfigFile == "" {
		return nil, nil //nolint:nilnil // no config means no inline config; caller treats nil as absent
	}

	data, err := os.ReadFile(filepath.Clean(opt.agentConfigFile))
	if err != nil {
		return nil, fmt.Errorf("failed to read agent config file: %w", err)
	}

	// An inline config requires a name (the server rejects spec-without-name), so derive a
	// stable one from the file name and fall back to "config" for an extension-only path.
	base := filepath.Base(opt.agentConfigFile)
	configName := strings.TrimSuffix(base, filepath.Ext(base))

	if configName == "" {
		configName = "config"
	}

	//exhaustruct:ignore
	return &v1.AgentConfig{
		AgentRemoteConfigs: []v1.AgentGroupRemoteConfig{
			{
				AgentRemoteConfigName: &configName,
				AgentRemoteConfigSpec: &v1.AgentRemoteConfigSpec{
					Value:       string(data),
					ContentType: "text/yaml",
				},
			},
		},
	}, nil
}

//nolint:lll
type formattedAgentGroup struct {
	Namespace                        string            `json:"namespace"                        short:"namespace"           text:"namespace"           yaml:"namespace"`
	Name                             string            `json:"name"                             short:"name"                text:"name"                yaml:"name"`
	Attributes                       map[string]string `json:"attributes"                       short:"-"                   text:"-"                   yaml:"attributes"`
	IdentifyingAttributesSelector    map[string]string `json:"identifyingAttributesSelector"    short:"-"                   text:"-"                   yaml:"identifyingAttributesSelector"`
	NonIdentifyingAttributesSelector map[string]string `json:"nonIdentifyingAttributesSelector" short:"-"                   text:"-"                   yaml:"nonIdentifyingAttributesSelector"`
	DeletedAt                        *time.Time        `json:"deletedAt,omitempty"              short:"deletedAt,omitempty" text:"deletedAt,omitempty" yaml:"deletedAt,omitempty"`
}

func toFormattedAgentGroup(agentGroup *v1.AgentGroup) *formattedAgentGroup {
	return &formattedAgentGroup{
		Namespace:                        agentGroup.Metadata.Namespace,
		Name:                             agentGroup.Metadata.Name,
		Attributes:                       agentGroup.Metadata.Attributes,
		IdentifyingAttributesSelector:    agentGroup.Spec.Selector.IdentifyingAttributes,
		NonIdentifyingAttributesSelector: agentGroup.Spec.Selector.NonIdentifyingAttributes,
		DeletedAt:                        switchToNilIfZero(agentGroup.Metadata.DeletedAt),
	}
}

func switchToNilIfZero(t *v1.Time) *time.Time {
	if t.IsZero() {
		return nil
	}

	return &t.Time
}
