// Package remoteconfigschema implements 'opampctl generate remoteconfigschema', which
// builds a RemoteConfigSchema from an OTel Collector's `components` output (parsed from a
// running binary, a file, or stdin), so a schema can be seeded per distribution/version.
package remoteconfigschema

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

var (
	// ErrNoInput is returned when no components source is provided.
	ErrNoInput = errors.New("provide --binary-path, or --from <file> (use - for stdin)")
	// ErrNameRequired is returned when a name cannot be determined.
	ErrNameRequired = errors.New("--name is required when the components output has no buildinfo.command/version")
	// ErrUnsupportedFormat is returned for an output format other than yaml/json.
	ErrUnsupportedFormat = errors.New("unsupported output format (use yaml or json)")
)

// CommandOptions holds the flags for the generate remoteconfigschema command.
type CommandOptions struct {
	binaryPath       string
	from             string
	name             string
	namespace        string
	binary           string
	version          string
	componentConfigs string
	formatType       string
}

// NewCommand creates the 'opampctl generate remoteconfigschema' command.
func NewCommand() *cobra.Command {
	options := &CommandOptions{
		binaryPath:       "",
		from:             "",
		name:             "",
		namespace:        "default",
		binary:           "",
		version:          "",
		componentConfigs: "",
		formatType:       "yaml",
	}

	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "remoteconfigschema",
		Short: "Generate a RemoteConfigSchema from an OTel Collector's components output",
		Long: "Generate a RemoteConfigSchema (component catalog) for a collector build.\n" +
			"The catalog is read from `otelcol components` output, sourced either by running a\n" +
			"binary (--binary-path) or from a file/stdin (--from).\n" +
			"\n" +
			"  opampctl generate remoteconfigschema --binary-path ./otelcol-contrib > schema.yaml\n" +
			"  ./otelcol-contrib components | opampctl generate remoteconfigschema --from - > schema.yaml",
		RunE: options.Run,
	}

	cmd.Flags().StringVar(&options.binaryPath, "binary-path", "",
		"Path to a collector binary; runs '<binary> components' to read the catalog")
	cmd.Flags().StringVarP(&options.from, "from", "f", "",
		"Path to a file containing 'otelcol components' output (use - for stdin)")
	cmd.Flags().StringVar(&options.name, "name", "", "Schema name (default: <command>-<version>)")
	cmd.Flags().StringVarP(&options.namespace, "namespace", "n", "default", "Namespace")
	cmd.Flags().StringVar(&options.binary, "binary", "",
		"Distribution label override (default: buildinfo.command)")
	cmd.Flags().StringVar(&options.version, "version", "",
		"Version override (default: buildinfo.version)")
	cmd.Flags().StringVar(&options.componentConfigs, "component-configs", "",
		"Path to a JSON file of per-component config field schemas to merge into the catalog")
	cmd.Flags().StringVarP(&options.formatType, "output", "o", "yaml", "Output format (yaml, json)")

	return cmd
}

// Run reads the components output, builds a RemoteConfigSchema, and writes it to stdout.
func (o *CommandOptions) Run(cmd *cobra.Command, _ []string) error {
	data, err := o.readComponents(cmd)
	if err != nil {
		return err
	}

	collected, err := parseComponents(data)
	if err != nil {
		return err
	}

	err = o.loadComponentConfigs(collected)
	if err != nil {
		return err
	}

	schema, err := o.buildSchema(collected)
	if err != nil {
		return err
	}

	return render(cmd.OutOrStdout(), schema, o.formatType)
}

// loadComponentConfigs reads the optional --component-configs JSON file (a
// v1.ComponentConfigCatalog produced by the reflection-based generator) into collected.
func (o *CommandOptions) loadComponentConfigs(collected *collected) error {
	if o.componentConfigs == "" {
		return nil
	}

	data, err := os.ReadFile(filepath.Clean(o.componentConfigs))
	if err != nil {
		return fmt.Errorf("read %q: %w", o.componentConfigs, err)
	}

	var catalog v1.ComponentConfigCatalog

	err = json.Unmarshal(data, &catalog)
	if err != nil {
		return fmt.Errorf("decode component configs %q: %w", o.componentConfigs, err)
	}

	collected.ComponentConfigs = catalog

	return nil
}

func (o *CommandOptions) readComponents(cmd *cobra.Command) ([]byte, error) {
	switch {
	case o.binaryPath != "":
		return runComponents(cmd.Context(), o.binaryPath)
	case o.from == "-":
		data, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}

		return data, nil
	case o.from != "":
		data, err := os.ReadFile(filepath.Clean(o.from))
		if err != nil {
			return nil, fmt.Errorf("read %q: %w", o.from, err)
		}

		return data, nil
	default:
		return nil, ErrNoInput
	}
}

func (o *CommandOptions) buildSchema(collected *collected) (*v1.RemoteConfigSchema, error) {
	binary := lo.CoalesceOrEmpty(o.binary, collected.Command)
	version := lo.CoalesceOrEmpty(o.version, collected.Version)

	name := o.name
	if name == "" {
		if binary == "" || version == "" {
			return nil, ErrNameRequired
		}

		name = binary + "-" + version
	}

	//exhaustruct:ignore // server-managed fields (CreatedAt, Conditions) are left zero
	return &v1.RemoteConfigSchema{
		Kind:       v1.RemoteConfigSchemaKind,
		APIVersion: v1.APIVersion,
		Metadata: v1.RemoteConfigSchemaMetadata{ //exhaustruct:ignore
			Name:      name,
			Namespace: o.namespace,
		},
		Spec: v1.RemoteConfigSchemaSpec{
			Binary:           binary,
			Version:          version,
			Components:       collected.Components,
			ComponentConfigs: collected.ComponentConfigs,
		},
	}, nil
}

// runComponents executes '<binaryPath> components' and returns its stdout.
func runComponents(ctx context.Context, binaryPath string) ([]byte, error) {
	out, err := exec.CommandContext(ctx, binaryPath, "components").Output()
	if err != nil {
		return nil, fmt.Errorf("run %q components: %w", binaryPath, err)
	}

	return out, nil
}

func render(writer io.Writer, schema *v1.RemoteConfigSchema, format string) error {
	switch format {
	case "json":
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")

		err := encoder.Encode(schema)
		if err != nil {
			return fmt.Errorf("encode json: %w", err)
		}

		return nil
	case "yaml":
		data, err := toYAML(schema)
		if err != nil {
			return err
		}

		_, err = writer.Write(data)
		if err != nil {
			return fmt.Errorf("write output: %w", err)
		}

		return nil
	default:
		return ErrUnsupportedFormat
	}
}

// toYAML marshals via JSON first so the v1 types' json tags are honored (yaml.v3 does
// not read json tags), yielding YAML keys that round-trip through 'create -f'.
func toYAML(value any) ([]byte, error) {
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode json: %w", err)
	}

	var generic any

	err = json.Unmarshal(jsonBytes, &generic)
	if err != nil {
		return nil, fmt.Errorf("decode json: %w", err)
	}

	data, err := yaml.Marshal(generic)
	if err != nil {
		return nil, fmt.Errorf("encode yaml: %w", err)
	}

	return data, nil
}
