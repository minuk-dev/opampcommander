// Package agentpackage provides the create agentpackage command for opampctl.
package agentpackage

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

// CommandOptions contains the options for the create agentpackage command.
type CommandOptions struct {
	*config.GlobalConfig

	// Flags
	name        string
	attributes  map[string]string
	packageType string
	version     string
	downloadURL string
	contentHash string
	signature   string
	headers     map[string]string
	formatType  string

	// internal state
	client *client.Client
}

// NewCommand creates a new create agentpackage command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "agentpackage",
		Short: "create agentpackage",
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

	cmd.Flags().StringVar(&options.name, "name", "", "Name of the agent package (required)")
	cmd.Flags().StringToStringVar(&options.attributes, "attributes", nil, "Attributes of the agent package (key=value)")
	cmd.Flags().StringVar(&options.packageType, "package-type", "", "Type of the package (e.g., TopLevelPackageName)")
	cmd.Flags().StringVar(&options.version, "version", "", "Version of the package")
	cmd.Flags().StringVar(&options.downloadURL, "download-url", "", "URL to download the package")
	cmd.Flags().StringVar(&options.contentHash, "content-hash", "", "Content hash of the package")
	cmd.Flags().StringVar(&options.signature, "signature", "", "Signature of the package")
	cmd.Flags().StringToStringVar(&options.headers, "headers", nil, "HTTP headers for downloading (key=value)")
	cmd.Flags().StringVarP(&options.formatType, "output", "o", "text", "Output format (text, json, yaml)")

	cmd.MarkFlagRequired("name") //nolint:errcheck,gosec

	return cmd
}

// Prepare prepares the create agentpackage command.
func (opt *CommandOptions) Prepare(*cobra.Command, []string) error {
	client, err := clientutil.NewClient(opt.GlobalConfig)
	if err != nil {
		return fmt.Errorf("failed to create authenticated client: %w", err)
	}

	opt.client = client

	return nil
}

// Run executes the create agentpackage command.
func (opt *CommandOptions) Run(cmd *cobra.Command, _ []string) error {
	agentPackageService := opt.client.AgentPackageService

	//exhaustruct:ignore
	createRequest := &v1.AgentPackage{
		Metadata: v1.AgentPackageMetadata{
			Name:       opt.name,
			Attributes: opt.attributes,
		},
		Spec: v1.AgentPackageSpec{
			PackageType: opt.packageType,
			Version:     opt.version,
			DownloadURL: opt.downloadURL,
			ContentHash: []byte(opt.contentHash),
			Signature:   []byte(opt.signature),
			Headers:     opt.headers,
		},
	}

	agentPackage, err := agentPackageService.CreateAgentPackage(cmd.Context(), createRequest)
	if err != nil {
		return fmt.Errorf("failed to create agent package: %w", err)
	}

	err = formatter.Format(cmd.OutOrStdout(), toFormattedAgentPackage(agentPackage), formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return nil
}

//nolint:lll
type formattedAgentPackage struct {
	Name        string            `json:"name"                short:"name"                text:"name"                yaml:"name"`
	Attributes  map[string]string `json:"attributes"          short:"-"                   text:"-"                   yaml:"attributes"`
	PackageType string            `json:"packageType"         short:"type"                text:"packageType"         yaml:"packageType"`
	Version     string            `json:"version"             short:"version"             text:"version"             yaml:"version"`
	DownloadURL string            `json:"downloadUrl"         short:"-"                   text:"downloadUrl"         yaml:"downloadUrl"`
	DeletedAt   *time.Time        `json:"deletedAt,omitempty" short:"deletedAt,omitempty" text:"deletedAt,omitempty" yaml:"deletedAt,omitempty"`
}

func toFormattedAgentPackage(agentPackage *v1.AgentPackage) *formattedAgentPackage {
	return &formattedAgentPackage{
		Name:        agentPackage.Metadata.Name,
		Attributes:  agentPackage.Metadata.Attributes,
		PackageType: agentPackage.Spec.PackageType,
		Version:     agentPackage.Spec.Version,
		DownloadURL: agentPackage.Spec.DownloadURL,
		DeletedAt:   switchToNilIfZero(agentPackage.Metadata.DeletedAt),
	}
}

func switchToNilIfZero(t *v1.Time) *time.Time {
	if t.IsZero() {
		return nil
	}

	return &t.Time
}
