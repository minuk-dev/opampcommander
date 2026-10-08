// Package agentremoteconfig implements the 'opampctl get agentremoteconfig' command.
package agentremoteconfig

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
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/get/internal/selectorflags"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

var (
	// ErrCommandExecutionFailed is returned when the command execution fails.
	ErrCommandExecutionFailed = errors.New("command execution failed")
)

// CommandOptions contains the options for the agentremoteconfig command.
type CommandOptions struct {
	*config.GlobalConfig

	// flags
	formatType     string
	includeDeleted bool
	namespace      string
	allNamespaces  bool
	selectors      selectorflags.Flags

	// internal
	client *client.Client
}

// NewCommand creates a new agentremoteconfig command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "agentremoteconfig",
		Short: "agentremoteconfig",
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
	cmd.Flags().BoolVar(&options.includeDeleted, "include-deleted", false, "Include soft-deleted agent remote configs")
	cmd.Flags().StringVarP(&options.namespace, "namespace", "n", "default", "Namespace")
	cmd.Flags().BoolVarP(&options.allNamespaces, "all-namespaces", "A", false, "List resources across all namespaces")

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
	if len(args) == 0 {
		err := opt.List(cmd)
		if err != nil {
			return fmt.Errorf("list failed: %w", err)
		}

		return nil
	}

	names := args

	err := opt.Get(cmd, names)
	if err != nil {
		return fmt.Errorf("get failed: %w", err)
	}

	return nil
}

// List retrieves the list of agent remote configs.
func (opt *CommandOptions) List(cmd *cobra.Command) error {
	selectorOpts, err := opt.selectors.ListOptions()
	if err != nil {
		return fmt.Errorf("failed to list agent remote configs: %w", err)
	}

	listOpts := append([]client.ListOption{client.WithIncludeDeleted(opt.includeDeleted)}, selectorOpts...)

	var agentRemoteConfigs []v1.AgentRemoteConfig

	if opt.allNamespaces {
		agentRemoteConfigs, err = opt.listAllNamespaces(cmd, listOpts...)
	} else {
		agentRemoteConfigs, err = clientutil.ListAgentRemoteConfigFully(
			cmd.Context(), opt.client, opt.namespace, listOpts...,
		)
	}

	if err != nil {
		return fmt.Errorf("failed to list agent remote configs: %w", err)
	}

	displayedConfigs := make([]formattedAgentRemoteConfig, len(agentRemoteConfigs))
	for idx, agentRemoteConfig := range agentRemoteConfigs {
		displayedConfigs[idx] = opt.toFormattedAgentRemoteConfig(agentRemoteConfig)
	}

	err = formatter.Format(cmd.OutOrStdout(), displayedConfigs, formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format agentremoteconfig: %w", err)
	}

	return nil
}

// Get retrieves the agent remote config information for the given names.
func (opt *CommandOptions) Get(cmd *cobra.Command, ids []string) error {
	getOpts := []client.GetOption{client.WithGetIncludeDeleted(opt.includeDeleted)}

	items, lookupErr := getutil.Collect(cmd, "agentremoteconfig", ids, func(id string) (*v1.AgentRemoteConfig, error) {
		return opt.client.AgentRemoteConfigService.GetAgentRemoteConfig(cmd.Context(), opt.namespace, id, getOpts...)
	})

	displayed := lo.Map(items, func(item v1.AgentRemoteConfig, _ int) formattedAgentRemoteConfig {
		return opt.toFormattedAgentRemoteConfig(item)
	})

	err := formatter.Format(cmd.OutOrStdout(), displayed, formatter.FormatType(opt.formatType))
	if err != nil {
		err = fmt.Errorf("failed to format agent remote configs: %w", err)
	}

	return errors.Join(lookupErr, err)
}

func (opt *CommandOptions) listAllNamespaces(
	cmd *cobra.Command, listOpts ...client.ListOption,
) ([]v1.AgentRemoteConfig, error) {
	configs, err := clientutil.ListAcrossNamespaces(
		cmd.Context(), opt.client,
		func(ctx context.Context, namespace string) ([]v1.AgentRemoteConfig, error) {
			return clientutil.ListAgentRemoteConfigFully(ctx, opt.client, namespace, listOpts...)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list agent remote configs across all namespaces: %w", err)
	}

	return configs, nil
}

//nolint:lll
type formattedAgentRemoteConfig struct {
	Namespace   string            `json:"namespace"           short:"namespace"   text:"namespace"           yaml:"namespace"`
	Name        string            `json:"name"                short:"name"        text:"name"                yaml:"name"`
	Attributes  map[string]string `json:"attributes"          short:"-"           text:"-"                   yaml:"attributes"`
	ContentType string            `json:"contentType"         short:"contentType" text:"contentType"         yaml:"contentType"`
	CreatedAt   time.Time         `json:"createdAt"           short:"createdAt"   text:"createdAt"           yaml:"createdAt"`
	CreatedBy   string            `json:"createdBy"           short:"createdBy"   text:"createdBy"           yaml:"createdBy"`
	DeletedAt   *time.Time        `json:"deletedAt,omitempty" short:"-"           text:"deletedAt,omitempty" yaml:"deletedAt,omitempty"`
	DeletedBy   *string           `json:"deletedBy,omitempty" short:"-"           text:"deletedBy,omitempty" yaml:"deletedBy,omitempty"`
}

func extractConditionInfo(conditions []v1.Condition) (time.Time, string, *time.Time, *string) {
	var (
		createdAt time.Time
		createdBy string
		deletedAt *time.Time
		deletedBy *string
	)

	for _, condition := range conditions {
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

	return createdAt, createdBy, deletedAt, deletedBy
}

func (opt *CommandOptions) toFormattedAgentRemoteConfig(
	agentRemoteConfig v1.AgentRemoteConfig,
) formattedAgentRemoteConfig {
	// Extract timestamps from metadata first, then fallback to conditions
	createdAt := agentRemoteConfig.Metadata.CreatedAt.Time

	// Get createdBy and deletedAt/deletedBy from conditions (createdBy is not in metadata)
	condCreatedAt, createdBy, deletedAt, deletedBy := extractConditionInfo(agentRemoteConfig.Status.Conditions)

	// Fallback to condition's createdAt if metadata doesn't have it
	if createdAt.IsZero() {
		createdAt = condCreatedAt
	}

	return formattedAgentRemoteConfig{
		Namespace:   agentRemoteConfig.Metadata.Namespace,
		Name:        agentRemoteConfig.Metadata.Name,
		Attributes:  agentRemoteConfig.Metadata.Attributes,
		ContentType: agentRemoteConfig.Spec.ContentType,
		CreatedAt:   createdAt,
		CreatedBy:   createdBy,
		DeletedAt:   deletedAt,
		DeletedBy:   deletedBy,
	}
}
