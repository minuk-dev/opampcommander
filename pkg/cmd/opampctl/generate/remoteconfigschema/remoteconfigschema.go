// Package remoteconfigschema implements 'opampctl generate remoteconfigschema', which
// builds a RemoteConfigSchema for a collector build: either from a schema registry,
// which describes the components a published release ships and the settings they
// accept, or from a collector's own `components` output, which is what a custom
// distribution can report about itself.
package remoteconfigschema

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
)

var (
	// ErrNoInput is returned when no schema source is provided.
	ErrNoInput = errors.New("provide --distribution, --binary-path, or --from <file> (use - for stdin)")
	// ErrNameRequired is returned when a name cannot be determined.
	ErrNameRequired = errors.New("--name is required when the components output has no buildinfo.command/version")
	// ErrUnsupportedFormat is returned for an output format other than yaml/json.
	ErrUnsupportedFormat = errors.New("unsupported output format (use yaml or json)")
)

// CommandOptions holds the flags for the generate remoteconfigschema command.
type CommandOptions struct {
	distribution   string
	schemaLocation string
	binaryPath     string
	from           string
	name           string
	namespace      string
	binary         string
	version        string
	stripDocs      bool
	formatType     string
}

// NewCommand creates the 'opampctl generate remoteconfigschema' command.
func NewCommand() *cobra.Command {
	options := &CommandOptions{
		distribution:   "",
		schemaLocation: DefaultSchemaLocation,
		binaryPath:     "",
		from:           "",
		name:           "",
		namespace:      "default",
		binary:         "",
		version:        VersionLatest,
		stripDocs:      false,
		formatType:     "yaml",
	}

	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "remoteconfigschema",
		Short: "Generate a RemoteConfigSchema for a collector build",
		Long: "Generate a RemoteConfigSchema (component catalog) for a collector build.\n" +
			"\n" +
			"For a published release, read it from the schema registry (--distribution), which\n" +
			"describes the components the release ships and the settings each one accepts.\n" +
			"For a custom distribution, read it from the binary's own `components` output\n" +
			"(--binary-path, or --from for a file/stdin), which reports the components it has\n" +
			"but not their settings.\n" +
			"\n" +
			"  opampctl generate remoteconfigschema --distribution contrib > schema.yaml\n" +
			"  opampctl generate remoteconfigschema --distribution core --version v0.150.0 > schema.yaml\n" +
			"  opampctl generate remoteconfigschema --binary-path ./otelcol-custom > schema.yaml\n" +
			"  ./otelcol-custom components | opampctl generate remoteconfigschema --from - > schema.yaml",
		RunE: options.Run,
	}

	cmd.Flags().StringVarP(&options.distribution, "distribution", "d", "",
		"Collector distribution to read from the schema registry (core, contrib, k8s, otlp, ...)")
	cmd.Flags().StringVar(&options.schemaLocation, "schema-location", DefaultSchemaLocation,
		"Schema registry base URL, or a path to a local checkout of one")
	cmd.Flags().StringVar(&options.binaryPath, "binary-path", "",
		"Path to a collector binary; runs '<binary> components' to read the catalog")
	cmd.Flags().StringVarP(&options.from, "from", "f", "",
		"Path to a file containing 'otelcol components' output (use - for stdin)")
	cmd.Flags().StringVar(&options.name, "name", "", "Schema name (default: <command>-<version>)")
	cmd.Flags().StringVarP(&options.namespace, "namespace", "n", "default", "Namespace")
	cmd.Flags().StringVar(&options.binary, "binary", "",
		"Distribution label override (default: buildinfo.command)")
	cmd.Flags().StringVar(&options.version, "version", VersionLatest,
		"Collector version: which release to read from the registry, or an override for buildinfo.version")
	cmd.Flags().BoolVar(&options.stripDocs, "strip-docs", false,
		"Drop field documentation, keeping only what validation needs")
	cmd.Flags().StringVarP(&options.formatType, "output", "o", "yaml", "Output format (yaml, json)")

	return cmd
}

// Run reads the component catalog, builds a RemoteConfigSchema, and writes it to stdout.
func (o *CommandOptions) Run(cmd *cobra.Command, _ []string) error {
	collected, err := o.collect(cmd)
	if err != nil {
		return err
	}

	if o.stripDocs {
		stripDocs(collected.Components)
	}

	schema, err := o.buildSchema(collected)
	if err != nil {
		return err
	}

	return render(cmd.OutOrStdout(), schema, o.formatType)
}

// collect reads the component catalog from whichever source the flags select.
func (o *CommandOptions) collect(cmd *cobra.Command) (*collected, error) {
	if o.distribution != "" {
		return fetchFromRegistry(cmd.Context(), o.schemaLocation, o.distribution, o.version)
	}

	data, err := o.readComponents(cmd)
	if err != nil {
		return nil, err
	}

	return parseComponents(data)
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
	binary := lo.CoalesceOrEmpty(o.binary, collected.Command, binaryForDistribution(o.distribution))
	version := lo.CoalesceOrEmpty(versionOverride(o.version), collected.Version)

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
			Binary:     binary,
			Version:    version,
			Components: collected.Components,
		},
	}, nil
}

// versionOverride reads --version as an override of the version the source reports.
// "latest" is not one: it selects which release to read from the registry, and the
// release itself then says which version it is.
func versionOverride(version string) string {
	if version == VersionLatest {
		return ""
	}

	return strings.TrimPrefix(version, "v")
}

// binaryForDistribution names the binary a registry distribution builds: the upstream
// releases publish "otelcol" for core and "otelcol-<distribution>" for the rest.
func binaryForDistribution(distribution string) string {
	switch distribution {
	case "":
		return ""
	case "core":
		return "otelcol"
	default:
		return "otelcol-" + distribution
	}
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

// yamlIndent is the indentation of generated YAML. The schema library is large enough
// that the difference from yaml.v3's default of four spaces is measured in megabytes.
const yamlIndent = 2

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

	var buffer bytes.Buffer

	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(yamlIndent)

	err = encoder.Encode(generic)
	if err != nil {
		return nil, fmt.Errorf("encode yaml: %w", err)
	}

	err = encoder.Close()
	if err != nil {
		return nil, fmt.Errorf("encode yaml: %w", err)
	}

	return buffer.Bytes(), nil
}
