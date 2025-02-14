package get

import (
	"github.com/minuk-dev/opampcommander/internal/opampctl/config"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/get/agent"
	"github.com/spf13/cobra"
)

type CommandOptions struct {
	*config.GlobalConfig
}

func NewCommand(options CommandOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "get",
	}

	cmd.AddCommand(agent.NewCommand(agent.CommandOptions{
		GlobalConfig: options.GlobalConfig,
	}))

	return cmd
}
