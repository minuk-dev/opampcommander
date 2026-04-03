// Package rbac provides the rbac command for opampctl.
package rbac

import (
	"github.com/spf13/cobra"

	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/rbac/assign"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/rbac/check"
	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/rbac/unassign"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

// CommandOptions contains the options for the rbac command.
type CommandOptions struct {
	*config.GlobalConfig
}

// NewCommand creates a new rbac command.
// It contains subcommands for RBAC operations.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "rbac",
		Short: "Manage RBAC (role assignments and permissions)",
	}

	cmd.AddCommand(assign.NewCommand(assign.CommandOptions{
		GlobalConfig: options.GlobalConfig,
	}))
	cmd.AddCommand(unassign.NewCommand(unassign.CommandOptions{
		GlobalConfig: options.GlobalConfig,
	}))
	cmd.AddCommand(check.NewCommand(check.CommandOptions{
		GlobalConfig: options.GlobalConfig,
	}))

	return cmd
}
