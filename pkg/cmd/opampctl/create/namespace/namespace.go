// Package namespace provides the command to create a namespace.
package namespace

import (
	"fmt"

	"github.com/spf13/cobra"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

// CommandOptions contains the options for the namespace create command.
type CommandOptions struct {
	*config.GlobalConfig

	formatType string
	client     *client.Client
}

// NewCommand creates a new create namespace command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "namespace [name]",
		Short: "Create a namespace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := options.Prepare(cmd, args)
			if err != nil {
				return err
			}

			return options.Run(cmd, args)
		},
	}
	cmd.Flags().StringVarP(
		&options.formatType, "output", "o", "short",
		"Output format (short, text, json, yaml)",
	)

	return cmd
}

// Prepare prepares the command to run.
func (opt *CommandOptions) Prepare(
	_ *cobra.Command, _ []string,
) error {
	cli, err := clientutil.NewClient(opt.GlobalConfig)
	if err != nil {
		return fmt.Errorf(
			"failed to create authenticated client: %w", err,
		)
	}

	opt.client = cli

	return nil
}

// Run creates a namespace.
func (opt *CommandOptions) Run(
	cmd *cobra.Command, args []string,
) error {
	name := args[0]

	//exhaustruct:ignore
	req := &v1.Namespace{
		Metadata: v1.NamespaceMetadata{
			Name: name,
		},
	}

	created, err := opt.client.NamespaceService.CreateNamespace(
		cmd.Context(), req,
	)
	if err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	formatType := formatter.FormatType(opt.formatType)

	switch formatType {
	case formatter.JSON, formatter.YAML:
		err = formatter.Format(
			cmd.OutOrStdout(), created, formatType,
		)
	case formatter.SHORT, formatter.TEXT:
		cmd.Printf("namespace/%s created\n", created.Metadata.Name)
	default:
		cmd.Printf("namespace/%s created\n", created.Metadata.Name)
	}

	if err != nil {
		return fmt.Errorf("failed to format namespace: %w", err)
	}

	return nil
}
