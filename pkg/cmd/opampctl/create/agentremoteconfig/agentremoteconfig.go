// Package agentremoteconfig provides the create agentremoteconfig command for opampctl.
package agentremoteconfig

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

// CommandOptions contains the options for the create agentremoteconfig command.
type CommandOptions struct {
	*config.GlobalConfig

	// Flags
	name        string
	attributes  map[string]string
	value       string
	file        string
	contentType string
	formatType  string

	// internal state
	client *client.Client
}

// NewCommand creates a new create agentremoteconfig command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "agentremoteconfig",
		Short: "create agentremoteconfig",
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

	cmd.Flags().StringVar(&options.name, "name", "", "Name of the agent remote config (required)")
	cmd.Flags().StringToStringVar(
		&options.attributes, "attributes", nil, "Attributes of the agent remote config (key=value)")
	cmd.Flags().StringVar(&options.value, "value", "", "Configuration value (alternative to --file)")
	cmd.Flags().StringVarP(&options.file, "file", "f", "", "Path to configuration file (alternative to --value)")
	cmd.Flags().StringVar(
		&options.contentType, "content-type", "",
		"Content type of the configuration (auto-detected from file extension if .yaml/.yml/.json)")
	cmd.Flags().StringVarP(&options.formatType, "output", "o", "text", "Output format (text, json, yaml)")

	cmd.MarkFlagRequired("name") //nolint:errcheck,gosec

	return cmd
}

// Prepare prepares the create agentremoteconfig command.
func (opt *CommandOptions) Prepare(*cobra.Command, []string) error {
	client, err := clientutil.NewClient(opt.GlobalConfig)
	if err != nil {
		return fmt.Errorf("failed to create authenticated client: %w", err)
	}

	opt.client = client

	return nil
}

// Run executes the create agentremoteconfig command.
func (opt *CommandOptions) Run(cmd *cobra.Command, _ []string) error {
	agentRemoteConfigService := opt.client.AgentRemoteConfigService

	valueContent, err := opt.loadValueContent()
	if err != nil {
		return err
	}

	contentType, err := opt.resolveContentType()
	if err != nil {
		return err
	}

	//exhaustruct:ignore
	createRequest := &v1.AgentRemoteConfig{
		Metadata: v1.AgentRemoteConfigMetadata{
			Name:       opt.name,
			Attributes: opt.attributes,
		},
		Spec: v1.AgentRemoteConfigSpec{
			Value:       valueContent,
			ContentType: contentType,
		},
	}

	agentRemoteConfig, err := agentRemoteConfigService.CreateAgentRemoteConfig(cmd.Context(), createRequest)
	if err != nil {
		return fmt.Errorf("failed to create agent remote config: %w", err)
	}

	formatted := toFormattedAgentRemoteConfig(agentRemoteConfig)

	err = formatter.Format(
		cmd.OutOrStdout(), formatted, formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return nil
}

//nolint:lll
type formattedAgentRemoteConfig struct {
	Name        string            `json:"name"                short:"name"                text:"name"                yaml:"name"`
	Attributes  map[string]string `json:"attributes"          short:"-"                   text:"-"                   yaml:"attributes"`
	ContentType string            `json:"contentType"         short:"contentType"         text:"contentType"         yaml:"contentType"`
	CreatedAt   time.Time         `json:"createdAt"           short:"createdAt"           text:"createdAt"           yaml:"createdAt"`
	CreatedBy   string            `json:"createdBy"           short:"createdBy"           text:"createdBy"           yaml:"createdBy"`
	DeletedAt   *time.Time        `json:"deletedAt,omitempty" short:"deletedAt,omitempty" text:"deletedAt,omitempty" yaml:"deletedAt,omitempty"`
	DeletedBy   *string           `json:"deletedBy,omitempty" short:"deletedBy,omitempty" text:"deletedBy,omitempty" yaml:"deletedBy,omitempty"`
}

func (opt *CommandOptions) loadValueContent() (string, error) {
	if opt.file != "" {
		content, err := os.ReadFile(opt.file)
		if err != nil {
			return "", fmt.Errorf("failed to read file: %w", err)
		}

		return string(content), nil
	}

	if opt.value == "" {
		return "", fmt.Errorf("either --value or --file must be specified")
	}

	return opt.value, nil
}

func (opt *CommandOptions) resolveContentType() (string, error) {
	if opt.contentType != "" {
		return opt.contentType, nil
	}

	if opt.file == "" {
		return "", fmt.Errorf("--content-type is required when using --value")
	}

	ext := filepath.Ext(opt.file)
	switch ext {
	case ".yaml", ".yml":
		return "application/yaml", nil
	case ".json":
		return "application/json", nil
	default:
		return "", fmt.Errorf("--content-type is required for file extension %q", ext)
	}
}

func toFormattedAgentRemoteConfig(agentRemoteConfig *v1.AgentRemoteConfig) *formattedAgentRemoteConfig {
	// Extract timestamps and users from conditions
	var (
		createdAt time.Time
		createdBy string
		deletedAt *time.Time
		deletedBy *string
	)

	for _, condition := range agentRemoteConfig.Status.Conditions {
		switch condition.Type { //nolint:exhaustive // Only handle Created and Deleted conditions
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

	return &formattedAgentRemoteConfig{
		Name:        agentRemoteConfig.Metadata.Name,
		Attributes:  agentRemoteConfig.Metadata.Attributes,
		ContentType: agentRemoteConfig.Spec.ContentType,
		CreatedAt:   createdAt,
		CreatedBy:   createdBy,
		DeletedAt:   deletedAt,
		DeletedBy:   deletedBy,
	}
}
