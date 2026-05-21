// Package agentpackage provides the create agentpackage command for opampctl.
package agentpackage

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

// CommandOptions contains the options for the create agentpackage command.
type CommandOptions struct {
	*config.GlobalConfig

	// Flags
	name        string
	namespace   string
	attributes  map[string]string
	packageType string
	version     string
	downloadURL string
	contentHash string
	signature   string
	headers     map[string]string
	file        string
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
	cmd.Flags().StringVarP(&options.namespace, "namespace", "n", "default", "Namespace of the agent package")
	cmd.Flags().StringToStringVar(&options.attributes, "attributes", nil, "Attributes of the agent package (key=value)")
	cmd.Flags().StringVar(&options.packageType, "package-type", "", "Type of the package (e.g., TopLevelPackageName)")
	cmd.Flags().StringVar(&options.version, "version", "", "Version of the package")
	cmd.Flags().StringVar(&options.downloadURL, "download-url", "", "URL to download the package")
	cmd.Flags().StringVar(&options.contentHash, "content-hash", "", "Content hash of the package")
	cmd.Flags().StringVar(&options.signature, "signature", "", "Signature of the package")
	cmd.Flags().StringToStringVar(&options.headers, "headers", nil, "HTTP headers for downloading (key=value)")
	cmd.Flags().StringVarP(&options.formatType, "output", "o", "text", "Output format (text, json, yaml)")
	cmd.Flags().StringVarP(&options.file, "file", "f", "",
		"Path to a full AgentPackage YAML definition. When set, individual CLI flags are ignored.")

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
	createRequest, namespace, err := opt.buildRequest()
	if err != nil {
		return err
	}

	agentPackage, err := opt.client.AgentPackageService.CreateAgentPackage(cmd.Context(), namespace, createRequest)
	if err != nil {
		return fmt.Errorf("failed to create agent package: %w", err)
	}

	err = formatter.Format(cmd.OutOrStdout(), toFormattedAgentPackage(agentPackage), formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	return nil
}

func (opt *CommandOptions) buildRequest() (*v1.AgentPackage, string, error) {
	if opt.file != "" {
		//exhaustruct:ignore
		req := &v1.AgentPackage{}

		err := yamlfile.Load(opt.file, req)
		if err != nil {
			return nil, "", fmt.Errorf("load agent package from %s: %w", opt.file, err)
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
	return &v1.AgentPackage{
		Metadata: v1.AgentPackageMetadata{
			Name:       opt.name,
			Namespace:  opt.namespace,
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
	}, opt.namespace, nil
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
