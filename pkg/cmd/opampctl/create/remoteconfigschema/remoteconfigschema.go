// Package remoteconfigschema provides the create remoteconfigschema command for opampctl.
package remoteconfigschema

import (
	"errors"
	"fmt"
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

// CommandOptions contains the options for the create remoteconfigschema command.
type CommandOptions struct {
	*config.GlobalConfig

	// Flags
	name       string
	namespace  string
	attributes map[string]string
	binary     string
	version    string
	file       string
	formatType string

	// internal state
	client *client.Client
}

// NewCommand creates a new create remoteconfigschema command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "remoteconfigschema",
		Short: "create remoteconfigschema",
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

	cmd.Flags().StringVar(&options.name, "name", "", "Name of the schema (required unless --file is used)")
	cmd.Flags().StringVarP(&options.namespace, "namespace", "n", "default", "Namespace")
	cmd.Flags().StringToStringVar(&options.attributes, "attributes", nil, "Attributes of the schema (key=value)")
	cmd.Flags().StringVar(&options.binary, "binary", "",
		"Collector build/distribution (e.g. otelcol, otelcol-contrib)")
	cmd.Flags().StringVar(&options.version, "version", "", "Collector version (semver) the schema describes")
	cmd.Flags().StringVarP(&options.file, "file", "f", "",
		"Path to a schema YAML/JSON definition (use to declare the component catalog)")
	cmd.Flags().StringVarP(&options.formatType, "output", "o", "text", "Output format (text, json, yaml)")

	return cmd
}

// Prepare prepares the create remoteconfigschema command.
func (opt *CommandOptions) Prepare(*cobra.Command, []string) error {
	client, err := clientutil.NewClient(opt.GlobalConfig)
	if err != nil {
		return fmt.Errorf("failed to create authenticated client: %w", err)
	}

	opt.client = client

	return nil
}

// Run executes the create remoteconfigschema command.
func (opt *CommandOptions) Run(cmd *cobra.Command, _ []string) error {
	createRequest, namespace, err := opt.buildRequest()
	if err != nil {
		return err
	}

	schema, err := opt.client.RemoteConfigSchemaService.CreateRemoteConfigSchema(cmd.Context(), namespace, createRequest)
	if err != nil {
		return fmt.Errorf("failed to create remote config schema: %w", err)
	}

	err = formatter.Format(cmd.OutOrStdout(), toFormatted(schema), formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return nil
}

func (opt *CommandOptions) buildRequest() (*v1.RemoteConfigSchema, string, error) {
	if opt.file != "" {
		//exhaustruct:ignore
		req := &v1.RemoteConfigSchema{}

		err := yamlfile.Load(opt.file, req)
		if err != nil {
			return nil, "", fmt.Errorf("load remote config schema from %s: %w", opt.file, err)
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

	//exhaustruct:ignore
	return &v1.RemoteConfigSchema{
		Metadata: v1.RemoteConfigSchemaMetadata{
			Name:       opt.name,
			Namespace:  opt.namespace,
			Attributes: opt.attributes,
		},
		Spec: v1.RemoteConfigSchemaSpec{
			Binary:           opt.binary,
			Version:          opt.version,
			Components:       v1.ComponentCatalog{},
			ComponentConfigs: nil,
		},
	}, opt.namespace, nil
}

type formattedRemoteConfigSchema struct {
	Name       string            `json:"name"       short:"name"      text:"name"      yaml:"name"`
	Attributes map[string]string `json:"attributes" short:"-"         text:"-"         yaml:"attributes"`
	Binary     string            `json:"binary"     short:"binary"    text:"binary"    yaml:"binary"`
	Version    string            `json:"version"    short:"version"   text:"version"   yaml:"version"`
	CreatedAt  time.Time         `json:"createdAt"  short:"createdAt" text:"createdAt" yaml:"createdAt"`
	CreatedBy  string            `json:"createdBy"  short:"createdBy" text:"createdBy" yaml:"createdBy"`
}

func toFormatted(schema *v1.RemoteConfigSchema) *formattedRemoteConfigSchema {
	var (
		createdAt time.Time
		createdBy string
	)

	for _, condition := range schema.Status.Conditions {
		if condition.Type == v1.ConditionTypeCreated && condition.Status == v1.ConditionStatusTrue {
			createdAt = condition.LastTransitionTime.Time
			createdBy = condition.Reason
		}
	}

	if createdAt.IsZero() {
		createdAt = schema.Metadata.CreatedAt.Time
	}

	return &formattedRemoteConfigSchema{
		Name:       schema.Metadata.Name,
		Attributes: schema.Metadata.Attributes,
		Binary:     schema.Spec.Binary,
		Version:    schema.Spec.Version,
		CreatedAt:  createdAt,
		CreatedBy:  createdBy,
	}
}
