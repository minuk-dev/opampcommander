package connection

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/spf13/cobra"

	v1connection "github.com/minuk-dev/opampcommander/api/v1/connection"
	"github.com/minuk-dev/opampcommander/internal/opampctl/config"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
)

type CommandOptions struct {
	*config.GlobalConfig

	// internal
	client *client.Client
}

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

	return cmd
}

func (opt *CommandOptions) Prepare(_ *cobra.Command, _ []string) error {
	opt.client = client.NewClient(opt.Endpoint)

	return nil
}

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

func (opt *CommandOptions) List(cmd *cobra.Command) error {
	connections, err := opt.client.ListConnections()
	if err != nil {
		return fmt.Errorf("failed to list agents: %w", err)
	}

	err = formatter.FormatYAML(cmd.OutOrStdout(), connections)
	if err != nil {
		return fmt.Errorf("failed to format yaml: %w", err)
	}

	return nil
}

func (opt *CommandOptions) Get(cmd *cobra.Command, ids []string) error {
	connections := make([]*v1connection.Connection, 0, len(ids))
	connectionIDs := lo.Map(ids, func(id string, _ int) uuid.UUID {
		connectionID, _ := uuid.Parse(id)

		return connectionID
	})

	for _, connectionID := range connectionIDs {
		connection, err := opt.client.GetConnection(connectionID)
		if err != nil {
			return fmt.Errorf("failed to get agent: %w", err)
		}

		connections = append(connections, connection)
	}

	cmd.Println(connections)

	return nil
}
