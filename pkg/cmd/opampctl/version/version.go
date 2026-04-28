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

const unavailable = "unavailable"

// Run executes the command.
func (opt *CommandOptions) Run(cmd *cobra.Command, _ []string) error {
	versionInfo, serverErr := opt.fetchVersionInfo(cmd)
	if serverErr != nil {
		cmd.PrintErrf("Failed to get server version: %v\n", serverErr)
	}

	err := opt.printVersion(cmd, versionInfo, serverErr)
	if err != nil {
		return fmt.Errorf("failed to format version information: %w", err)
	}

	return nil
}

func (opt *CommandOptions) fetchVersionInfo(cmd *cobra.Command) (Version, error) {
	v := version.Get()

	//exhaustruct:ignore
	versionInfo := Version{ClientVersion: &v}

	if opt.clientOnly {
		return versionInfo, nil
	}

	serverVersion, err := opt.client.GetServerVersion(cmd.Context())
	if err != nil {
		//exhaustruct:ignore
		versionInfo.ServerVersion = &v1version.Info{}

		return versionInfo, fmt.Errorf("failed to get server version: %w", err)
	}

	versionInfo.ServerVersion = serverVersion

	return versionInfo, nil
}

func serverVersionOrUnavailable(v string, serverErr error) string {
	if v == "" && serverErr != nil {
		return unavailable
	}

	return v
}

func (opt *CommandOptions) printVersion(cmd *cobra.Command, versionInfo Version, serverErr error) error {
	switch opt.formatType {
	case "short":
		type shortItem struct {
			ClientGitVersion string `short:"clientGitVersion"`
			ServerGitVersion string `short:"serverGitVersion"`
		}

		err := formatter.FormatShort(cmd.OutOrStdout(), shortItem{
			ClientGitVersion: versionInfo.ClientVersion.GitVersion,
			ServerGitVersion: serverVersionOrUnavailable(versionInfo.ServerVersion.GitVersion, serverErr),
		})
		if err != nil {
			return fmt.Errorf("failed to format short: %w", err)
		}

		return nil
	case "text":
		type textItem struct {
			ClientGitVersion string `text:"clientGitVersion"`
			ClientBuildDate  string `text:"clientBuildDate"`
			ClientCommitHash string `text:"clientCommitHash"`
			ServerGitVersion string `text:"serverGitVersion"`
			ServerBuildDate  string `text:"serverBuildDate"`
			ServerCommitHash string `text:"serverCommitHash"`
		}

		err := formatter.FormatText(cmd.OutOrStdout(), textItem{
			ClientGitVersion: versionInfo.ClientVersion.GitVersion,
			ClientBuildDate:  versionInfo.ClientVersion.BuildDate,
			ClientCommitHash: versionInfo.ClientVersion.GitCommit,
			ServerGitVersion: serverVersionOrUnavailable(versionInfo.ServerVersion.GitVersion, serverErr),
			ServerBuildDate:  versionInfo.ServerVersion.BuildDate,
			ServerCommitHash: versionInfo.ServerVersion.GitCommit,
		})
		if err != nil {
			return fmt.Errorf("failed to format text: %w", err)
		}

		return nil
	default:
		err := formatter.Format(cmd.OutOrStdout(), versionInfo, formatter.FormatType(opt.formatType))
		if err != nil {
			return fmt.Errorf("failed to format: %w", err)
		}

		return nil
	}
}
