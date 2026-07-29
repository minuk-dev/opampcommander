// Package generate provides the 'opampctl generate' command group, which produces
// resource definitions locally (no server connection) for piping into 'create -f'.
package generate

import (
	"github.com/spf13/cobra"

	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/generate/remoteconfigschema"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

// CommandOptions contains the options for the generate command.
type CommandOptions struct {
	*config.GlobalConfig
}

// NewCommand creates a new generate command. Its subcommands generate resource
// definitions offline and therefore do not need the global config or a server.
func NewCommand(_ CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate resource definitions to stdout",
		Long: "Generate ready-to-apply resource definitions to stdout.\n" +
			"Redirect the output to a file, review it, then create the resource:\n" +
			"\n" +
			"  opampctl generate remoteconfigschema --binary-path ./otelcol-contrib > schema.yaml\n" +
			"  opampctl create remoteconfigschema -f schema.yaml",
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			// Generation does not need the global config or a server.
			return nil
		},
	}

	cmd.AddCommand(remoteconfigschema.NewCommand())

	return cmd
}
