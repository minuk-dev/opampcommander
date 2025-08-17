package version

import (
	"fmt"

	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
	"github.com/minuk-dev/opampcommander/pkg/version"
	"github.com/spf13/cobra"

	v1version "github.com/minuk-dev/opampcommander/api/v1/version"
)

// CommandOptions contains the options for the version command.
type CommandOptions struct {
	*config.GlobalConfig

	// flags
	clientOnly bool
	formatType string

	// internal
	client *client.Client
}

type Version struct {
	ClientVersion *v1version.Info `json:"clientVersion" yaml:"clientVersion" short:"client"`
	ServerVersion *v1version.Info `json:"serverVersion" yaml:"serverVersion" short:"server"`
}

// NewCommand creates a new version command.
func NewCommand(options CommandOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version of opampctl",
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
	cmd.Flags().BoolVar(&options.clientOnly, "client-only", false, "Print only the client version without connecting to the server")
	cmd.Flags().StringVarP(&options.formatType, "output", "o", "short", "Output format (short, text, json, yaml)")

	return cmd
}

// Prepare prepares the command to run.
func (opt *CommandOptions) Prepare(cmd *cobra.Command, args []string) error {
	client := clientutil.NewUnauthenticatedClient(opt.GlobalConfig)

	opt.client = client

	return nil
}

// Run executes the command.
func (opt *CommandOptions) Run(cmd *cobra.Command, args []string) error {
	var (
		serverErr   error
		versionInfo Version
	)

	versionInfo.ClientVersion = func() *v1version.Info { v := version.Get(); return &v }()
	if !opt.clientOnly {
		versionInfo.ServerVersion, serverErr = opt.client.GetServerVersion(cmd.Context())
		if serverErr != nil {
			cmd.PrintErrf("Failed to get server version: %v\n", serverErr)
		}
	}

	err := formatter.Format(cmd.OutOrStdout(), versionInfo, formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format version information: %w", err)
	}
	return nil
}
