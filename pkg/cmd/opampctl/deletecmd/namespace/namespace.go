// Package namespace provides the command to delete a namespace.
package namespace

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

// CommandOptions contains the options for the namespace delete command.
type CommandOptions struct {
	*config.GlobalConfig

	client *client.Client
}

// NewCommand creates a new delete namespace command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "namespace [name]",
		Short: "Delete a namespace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := options.Prepare(cmd, args)
			if err != nil {
				return err
			}

			return options.Run(cmd, args)
		},
	}

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

// Run deletes a namespace.
func (opt *CommandOptions) Run(
	cmd *cobra.Command, args []string,
) error {
	name := args[0]

	err := opt.client.NamespaceService.DeleteNamespace(
		cmd.Context(), name,
	)
	if err != nil {
		return fmt.Errorf("failed to delete namespace: %w", err)
	}

	cmd.Printf("namespace/%s deleted\n", name)

	return nil
}
