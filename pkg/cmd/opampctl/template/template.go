// Package template provides the template command for opampctl.
// It prints example resource definitions to stdout so users can redirect
// the output to a file, customize it, and then create the resource.
package template

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

const outputFlag = "output"

// ErrUnknownExample is returned when the requested example name is not registered.
var ErrUnknownExample = errors.New("unknown example")

// ErrUnsupportedFormat is returned when an output format other than yaml/json is requested.
var ErrUnsupportedFormat = errors.New("unsupported output format")

// CommandOptions contains the options for the template command.
type CommandOptions struct {
	*config.GlobalConfig
}

// NewCommand creates a new template command.
// The template command and its subcommands do not require the global config
// or a server connection — they only emit static example definitions.
func NewCommand(_ CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Print example resource templates to stdout",
		Long: "Print ready-to-edit example resource definitions to stdout.\n" +
			"Redirect the output to a file, customize it, then create the resource:\n" +
			"\n" +
			"  opampctl template agentgroup basic > my-agentgroup.yaml",
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			// Template generation does not need the global config or a server.
			return nil
		},
	}

	cmd.PersistentFlags().StringP(outputFlag, "o", "yaml",
		"Output format (yaml preserves comments; json does not)")

	cmd.AddCommand(newAgentGroupCommand())
	cmd.AddCommand(newAgentPackageCommand())
	cmd.AddCommand(newAgentRemoteConfigCommand())
	cmd.AddCommand(newCertificateCommand())
	cmd.AddCommand(newNamespaceCommand())
	cmd.AddCommand(newRoleCommand())
	cmd.AddCommand(newRoleBindingCommand())

	return cmd
}

// newKindCommand wires a kind-specific template subcommand using the shared
// implementation. Each kind only needs to provide its set of examples.
func newKindCommand(use, short string, examples map[string]string) *cobra.Command {
	//exhaustruct:ignore
	return &cobra.Command{
		Use:   use + " [example]",
		Short: short,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, args, examples)
		},
	}
}

func run(cmd *cobra.Command, args []string, examples map[string]string) error {
	names := slices.Sorted(maps.Keys(examples))

	if len(args) == 0 {
		printList(cmd.OutOrStdout(), names, cmd.CommandPath())

		return nil
	}

	body, ok := examples[args[0]]
	if !ok {
		return fmt.Errorf("%w %q; available: %s", ErrUnknownExample, args[0], strings.Join(names, ", "))
	}

	return write(cmd.OutOrStdout(), body, lookupFormat(cmd))
}

func lookupFormat(cmd *cobra.Command) string {
	flag := cmd.Flag(outputFlag)
	if flag == nil {
		return "yaml"
	}

	return flag.Value.String()
}

func printList(writer io.Writer, names []string, cmdPath string) {
	fmt.Fprintf(writer, "Available examples for %s:\n", cmdPath) //nolint:errcheck

	for _, name := range names {
		fmt.Fprintf(writer, "  %s\n", name) //nolint:errcheck
	}

	fmt.Fprintf(writer, "\nRun \"%s <example>\" to print one to stdout.\n", cmdPath) //nolint:errcheck
}

func write(writer io.Writer, body, format string) error {
	switch format {
	case "yaml", "":
		_, err := io.WriteString(writer, body)
		if err != nil {
			return fmt.Errorf("failed to write yaml: %w", err)
		}

		return nil
	case "json":
		var data any

		err := yaml.Unmarshal([]byte(body), &data)
		if err != nil {
			return fmt.Errorf("failed to parse template as yaml: %w", err)
		}

		encoded, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal json: %w", err)
		}

		_, err = writer.Write(append(encoded, '\n'))
		if err != nil {
			return fmt.Errorf("failed to write json: %w", err)
		}

		return nil
	default:
		return fmt.Errorf("%w %q (supported: yaml, json)", ErrUnsupportedFormat, format)
	}
}
