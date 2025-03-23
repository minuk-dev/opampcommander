package opampctl

import (
	"github.com/spf13/cobra"

	"github.com/minuk-dev/opampcommander/internal/opampctl/config"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/get"
)

type CommandOption struct {
	// flags
	*config.GlobalConfig
}

func NewCommand(options CommandOption) *cobra.Command {
	if options.GlobalConfig == nil {
		options.GlobalConfig = &config.GlobalConfig{
			Endpoint: "http://localhost:8080",
		}
	}
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "opampctl",
		Short: "opampctl",
	}
	cmd.PersistentFlags().StringVar(&options.Endpoint, "endpoint", "http://localhost:8080", "opampcommander endpoint")
	cmd.AddCommand(get.NewCommand(get.CommandOptions{
		GlobalConfig: options.GlobalConfig,
	}))

	return cmd
}
