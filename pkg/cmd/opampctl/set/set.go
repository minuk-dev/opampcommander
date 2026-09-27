// Package set provides the set command for opampctl.
package set

import (
	"github.com/spf13/cobra"

	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/set/instanceuid"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/set/opampcertificate"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

// NewCommand creates a new set command.
func NewCommand(globalConfig *config.GlobalConfig) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set specific features on resources",
	}

	cmd.AddCommand(instanceuid.NewCommand(instanceuid.CommandOptions{
		GlobalConfig: globalConfig,
	}))
	cmd.AddCommand(opampcertificate.NewCommand(opampcertificate.CommandOptions{GlobalConfig: globalConfig}))

	return cmd
}
