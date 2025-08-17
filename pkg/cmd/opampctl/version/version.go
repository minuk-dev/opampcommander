// Package version provides a command to print the version of opampctl.
package version

import (
	"fmt"

	"github.com/spf13/cobra"

	v1version "github.com/minuk-dev/opampcommander/api/v1/version"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
	"github.com/minuk-dev/opampcommander/pkg/version"
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

// Version is a struct that contains the client and server version information.
type Version struct {
	ClientVersion *v1version.Info `json:"clientVersion" short:"client" yaml:"clientVersion"`
	ServerVersion *v1version.Info `json:"serverVersion" short:"server" yaml:"serverVersion"`
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
	//nolint:lll
	cmd.Flags().BoolVar(&options.clientOnly, "client-only", false, "Print only the client version without connecting to the server")
	cmd.Flags().StringVarP(&options.formatType, "output", "o", "short", "Output format (short, text, json, yaml)")

	return cmd
}

// Prepare prepares the command to run.
func (opt *CommandOptions) Prepare(*cobra.Command, []string) error {
	client := clientutil.NewUnauthenticatedClient(opt.GlobalConfig)

	opt.client = client

	return nil
}

// Run executes the command.
func (opt *CommandOptions) Run(cmd *cobra.Command, _ []string) error {
	var (
		serverErr   error
		versionInfo Version
	)

	versionInfo.ClientVersion = func() *v1version.Info {
		v := version.Get()

		return &v
	}()
	if !opt.clientOnly {
		versionInfo.ServerVersion, serverErr = opt.client.GetServerVersion(cmd.Context())
		if serverErr != nil {
			cmd.PrintErrf("Failed to get server version: %v\n", serverErr)
		}
	}

	var formatErr error

	switch opt.formatType {
	case "short":
		type shortItem struct {
			ClientGitVersion string `json:"clientGitVersion" yaml:"clientGitVersion"`
			ServerGitVersion string `json:"serverGitVersion" yaml:"serverGitVersion"`
		}

		formatErr = formatter.FormatShort(cmd.OutOrStdout(), shortItem{
			ClientGitVersion: versionInfo.ClientVersion.GitVersion,
			ServerGitVersion: versionInfo.ServerVersion.GitVersion,
		})
	case "text":
		type textItem struct {
			ClientGitVersion string `json:"clientGitVersion" yaml:"clientGitVersion"`
			ClientBuildDate  string `json:"clientBuildDate"  yaml:"clientBuildDate"`
			ClientCommitHash string `json:"clientCommitHash" yaml:"clientCommitHash"`
			ServerGitVersion string `json:"serverGitVersion" yaml:"serverGitVersion"`
			ServerBuildDate  string `json:"serverBuildDate"  yaml:"serverBuildDate"`
			ServerCommitHash string `json:"serverCommitHash" yaml:"serverCommitHash"`
		}

		formatErr = formatter.FormatText(cmd.OutOrStdout(), textItem{
			ClientGitVersion: versionInfo.ClientVersion.GitVersion,
			ClientBuildDate:  versionInfo.ClientVersion.BuildDate,
			ClientCommitHash: versionInfo.ClientVersion.GitCommit,
			ServerGitVersion: versionInfo.ServerVersion.GitVersion,
			ServerBuildDate:  versionInfo.ServerVersion.BuildDate,
			ServerCommitHash: versionInfo.ServerVersion.GitCommit,
		})
	default:
		formatErr = formatter.Format(cmd.OutOrStdout(), versionInfo, formatter.FormatType(opt.formatType))
	}

	if formatErr != nil {
		return fmt.Errorf("failed to format version information: %w", formatErr)
	}

	return nil
}
