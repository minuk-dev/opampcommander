// Package connection provides the command to get connection information.
package connection

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/spf13/cobra"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
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
	var (
		connections []v1.Connection
		err         error
	)

	if opt.allNamespaces {
		connections, err = opt.listAllNamespaces(cmd)
	} else {
		connections, err = clientutil.ListConnectionFully(cmd.Context(), opt.client, opt.namespace)
	}

	if err != nil {
		return fmt.Errorf("failed to list connections: %w", err)
	}

	if len(connections) == 0 {
		cmd.Println("No connections found.")
	}

	err = formatter.Format(cmd.OutOrStdout(), connections, formatter.FormatType(opt.formatType))
	if err != nil {
		return fmt.Errorf("failed to format connections: %w", err)
	}

	return nil
}

// Get retrieves the connection information for the given IDs.
func (opt *CommandOptions) Get(cmd *cobra.Command, ids []string) error {
	connections := make([]*v1.Connection, 0, len(ids))
	connectionIDs := lo.Map(ids, func(id string, _ int) uuid.UUID {
		connectionID, _ := uuid.Parse(id)

		return connectionID
	})

	for _, connectionID := range connectionIDs {
		connection, err := opt.client.ConnectionService.GetConnection(cmd.Context(), opt.namespace, connectionID)
		if err != nil {
			return fmt.Errorf("failed to get agent: %w", err)
		}

		connections = append(connections, connection)
	}

	cmd.Println(connections)

	return nil
}

func (opt *CommandOptions) listAllNamespaces(cmd *cobra.Command) ([]v1.Connection, error) {
	connections, err := clientutil.ListAcrossNamespaces(
		cmd.Context(), opt.client,
		func(ctx context.Context, namespace string) ([]v1.Connection, error) {
			return clientutil.ListConnectionFully(ctx, opt.client, namespace)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list connections across all namespaces: %w", err)
	}

	return connections, nil
}
