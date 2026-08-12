// Package remoteconfigschema implements the 'opampctl get remoteconfigschema' command.
package remoteconfigschema

import (
	"context"
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

// ErrCommandExecutionFailed is returned when the command execution fails.
var ErrCommandExecutionFailed = errors.New("command execution failed")

// CommandOptions contains the options for the remoteconfigschema command.
type CommandOptions struct {
	*config.GlobalConfig

	// flags
	formatType     string
	includeDeleted bool
	namespace      string
	allNamespaces  bool

	// internal
	client *client.Client
}

// NewCommand creates a new remoteconfigschema command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "remoteconfigschema",
		Short: "remoteconfigschema",
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
	cmd.Flags().BoolVar(&options.includeDeleted, "include-deleted", false, "Include soft-deleted schemas")
	cmd.Flags().StringVarP(&options.namespace, "namespace", "n", "default", "Namespace")
	cmd.Flags().BoolVarP(&options.allNamespaces, "all-namespaces", "A", false, "List resources across all namespaces")

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

	err := opt.Get(cmd, args)
	if err != nil {
		return fmt.Errorf("get failed: %w", err)
	}

	return nil
}

// List retrieves the list of schemas.
func (opt *CommandOptions) List(cmd *cobra.Command) error {
	listOpts := []client.ListOption{client.WithIncludeDeleted(opt.includeDeleted)}

	var (
		schemas []v1.RemoteConfigSchema
		err     error
	)

	if opt.allNamespaces {
		schemas, err = opt.listAllNamespaces(cmd, listOpts...)
	} else {
		schemas, err = clientutil.ListRemoteConfigSchemaFully(cmd.Context(), opt.client, opt.namespace, listOpts...)
	}

	if err != nil {
		return fmt.Errorf("failed to list remote config schemas: %w", err)
	}

	displayed := lo.Map(schemas, func(s v1.RemoteConfigSchema, _ int) formattedRemoteConfigSchema {
		return toFormatted(s)
	})

	err = formatter.Format(cmd.OutOrStdout(), displayed, formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format remote config schema: %w", err)
	}

	return nil
}

// Get retrieves the schema information for the given names.
func (opt *CommandOptions) Get(cmd *cobra.Command, names []string) error {
	type schemaWithErr struct {
		Schema *v1.RemoteConfigSchema
		Err    error
	}

	getOpts := []client.GetOption{client.WithGetIncludeDeleted(opt.includeDeleted)}

	schemasWithErr := lo.Map(names, func(name string, _ int) schemaWithErr {
		schema, err := opt.client.RemoteConfigSchemaService.GetRemoteConfigSchema(
			cmd.Context(), opt.namespace, name, getOpts...)

		return schemaWithErr{Schema: schema, Err: err}
	})

	schemas := lo.Filter(schemasWithErr, func(s schemaWithErr, _ int) bool {
		return s.Err == nil
	})
	if len(schemas) == 0 {
		cmd.Println("No remote config schemas found or all specified schemas could not be retrieved.")

		return nil
	}

	displayed := lo.Map(schemas, func(s schemaWithErr, _ int) formattedRemoteConfigSchema {
		return toFormatted(*s.Schema)
	})

	err := formatter.Format(cmd.OutOrStdout(), displayed, formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format remote config schemas: %w", err)
	}

	errs := lo.Filter(schemasWithErr, func(s schemaWithErr, _ int) bool {
		return s.Err != nil
	})
	if len(errs) > 0 {
		errMessages := lo.Map(errs, func(s schemaWithErr, _ int) string {
			return s.Err.Error()
		})

		cmd.PrintErrf("Some remote config schemas could not be retrieved: %s", strings.Join(errMessages, ", "))
	}

	return nil
}

func (opt *CommandOptions) listAllNamespaces(
	cmd *cobra.Command, listOpts ...client.ListOption,
) ([]v1.RemoteConfigSchema, error) {
	schemas, err := clientutil.ListAcrossNamespaces(
		cmd.Context(), opt.client,
		func(ctx context.Context, namespace string) ([]v1.RemoteConfigSchema, error) {
			return clientutil.ListRemoteConfigSchemaFully(ctx, opt.client, namespace, listOpts...)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list remote config schemas across all namespaces: %w", err)
	}

	return schemas, nil
}

//nolint:lll
type formattedRemoteConfigSchema struct {
	Namespace  string            `json:"namespace"           short:"namespace"  text:"namespace"           yaml:"namespace"`
	Name       string            `json:"name"                short:"name"       text:"name"                yaml:"name"`
	Binary     string            `json:"binary"              short:"binary"     text:"binary"              yaml:"binary"`
	Version    string            `json:"version"             short:"version"    text:"version"             yaml:"version"`
	Components int               `json:"components"          short:"components" text:"components"          yaml:"components"`
	Attributes map[string]string `json:"attributes"          short:"-"          text:"-"                   yaml:"attributes"`
	CreatedAt  time.Time         `json:"createdAt"           short:"createdAt"  text:"createdAt"           yaml:"createdAt"`
	CreatedBy  string            `json:"createdBy"           short:"createdBy"  text:"createdBy"           yaml:"createdBy"`
	DeletedAt  *time.Time        `json:"deletedAt,omitempty" short:"-"          text:"deletedAt,omitempty" yaml:"deletedAt,omitempty"`
	DeletedBy  *string           `json:"deletedBy,omitempty" short:"-"          text:"deletedBy,omitempty" yaml:"deletedBy,omitempty"`
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

// componentCount returns the total number of components across all classes.
func componentCount(catalog v1.ComponentCatalog) int {
	total := 0
	for _, components := range catalog {
		total += len(components)
	}

	return total
}

func toFormatted(schema v1.RemoteConfigSchema) formattedRemoteConfigSchema {
	createdAt := schema.Metadata.CreatedAt.Time

	condCreatedAt, createdBy, deletedAt, deletedBy := extractConditionInfo(schema.Status.Conditions)
	if createdAt.IsZero() {
		createdAt = condCreatedAt
	}

	return formattedRemoteConfigSchema{
		Namespace:  schema.Metadata.Namespace,
		Name:       schema.Metadata.Name,
		Binary:     schema.Spec.Binary,
		Version:    schema.Spec.Version,
		Components: componentCount(schema.Spec.Components),
		Attributes: schema.Metadata.Attributes,
		CreatedAt:  createdAt,
		CreatedBy:  createdBy,
		DeletedAt:  deletedAt,
		DeletedBy:  deletedBy,
	}
}
