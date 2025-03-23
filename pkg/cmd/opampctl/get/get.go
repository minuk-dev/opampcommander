package get

import (
	"github.com/spf13/cobra"

	"github.com/minuk-dev/opampcommander/internal/opampctl/config"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/get/agent"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/get/connection"
)

type CommandOptions struct {
	*config.GlobalConfig
}

func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "get",
		Short: "get",
	}

	cmd.AddCommand(agent.NewCommand(agent.CommandOptions{
		GlobalConfig: options.GlobalConfig,
	}))
	cmd.AddCommand(connection.NewCommand(connection.CommandOptions{
		GlobalConfig: options.GlobalConfig,
	}))

	return cmd
}
