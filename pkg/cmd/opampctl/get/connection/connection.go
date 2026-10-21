// Package connection provides the command to get connection information.
package connection

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/get/internal/getutil"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/get/internal/listutil"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

// CommandOptions contains the options for the connection command.
type CommandOptions struct {
	*config.GlobalConfig

	// flags
	formatType    string
	namespace     string
	allNamespaces bool

	pagination listutil.Flags

	// internal
	client *client.Client
}

// NewCommand creates a new connection command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "connection",
		Short: "connection",
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
	cmd.Flags().StringVarP(&options.formatType, "format", "f", "short", "Output format (short, text, json, yaml)")
	cmd.Flags().StringVarP(&options.namespace, "namespace", "n", "default", "Namespace")
	cmd.Flags().BoolVarP(&options.allNamespaces, "all-namespaces", "A", false, "List resources across all namespaces")

	options.pagination.Register(cmd)

	return cmd
}

// Prepare prepares the command.
func (opt *CommandOptions) Prepare(*cobra.Command, []string) error {
	config := opt.GlobalConfig

	client, err := clientutil.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to create authenticated client: %w", err)
	}

	opt.client = client

	return nil
}

// Run runs the command.
func (opt *CommandOptions) Run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		err := opt.List(cmd)
		if err != nil {
			return fmt.Errorf("list failed: %w", err)
		}

		return nil
	}

	agentUIDs := args

	err := opt.Get(cmd, agentUIDs)
	if err != nil {
		return fmt.Errorf("get failed: %w", err)
	}

	return nil
}

// List retrieves the connection information for all connections in a namespace.
func (opt *CommandOptions) List(cmd *cobra.Command) error {
	var listOpts []client.ListOption

	err := listutil.Run(cmd, opt.client, opt.pagination, opt.namespace, opt.allNamespaces,
		formatter.FormatType(opt.formatType),
		func(
			ctx context.Context, namespace string, pageOpts ...client.ListOption,
		) (*v1.ListResponse[v1.Connection], error) {
			return opt.client.ConnectionService.ListConnections(ctx, namespace, append(listOpts, pageOpts...)...)
		}, func(items []v1.Connection) any { return items })
	if err != nil {
		return fmt.Errorf("list connection: %w", err)
	}

	return nil
}

// Get retrieves the connection information for the given IDs.
func (opt *CommandOptions) Get(cmd *cobra.Command, ids []string) error {
	items, lookupErr := getutil.Collect(cmd, "connection", ids, func(id string) (*v1.Connection, error) {
		instanceUID, err := uuid.Parse(id)
		if err != nil {
			return nil, fmt.Errorf("invalid UID: %w", err)
		}

		return opt.client.ConnectionService.GetConnection(cmd.Context(), opt.namespace, instanceUID)
	})

	err := formatter.Format(cmd.OutOrStdout(), items, formatter.FormatType(opt.formatType))
	if err != nil {
		err = fmt.Errorf("failed to format connections: %w", err)
	}

	return errors.Join(lookupErr, err)
}
