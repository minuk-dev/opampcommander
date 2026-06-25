// Package reconcile provides the generic `opampctl reconcile` command. It re-enforces a
// domain object's invariants on demand by dispatching to the server's reconcile registry,
// so any reconcilable kind is reachable through a single command without per-kind code.
package reconcile

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

// reconcileArgs is the number of positional arguments: <kind> <name>.
const reconcileArgs = 2

// CommandOptions contains the options for the reconcile command.
type CommandOptions struct {
	GlobalConfig *config.GlobalConfig
}

// NewCommand creates a new reconcile command.
func NewCommand(options CommandOptions) *cobra.Command {
	var namespace string

	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "reconcile <kind> <name>",
		Short: "reconcile a resource",
		Long: "Re-run the side effects that normally fire when a resource is created/updated, " +
			"to re-enforce its domain rules on demand. For kind \"agent\", <name> is the instance UID.",
		Args: cobra.ExactArgs(reconcileArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := clientutil.NewClient(options.GlobalConfig)
			if err != nil {
				return fmt.Errorf("failed to create API client: %w", err)
			}

			kind, name := args[0], args[1]

			err = cli.ReconcileService.Reconcile(context.Background(), kind, namespace, name)
			if err != nil {
				return fmt.Errorf("failed to reconcile %s %q: %w", kind, name, err)
			}

			cmd.Printf("%s %s/%s reconciled successfully\n", kind, namespace, name)

			return nil
		},
		ValidArgsFunction: kindCompletion(options),
	}

	cmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "Namespace of the resource")

	return cmd
}

// kindCompletion completes the first argument (kind) from the server's reconcile registry.
func kindCompletion(
	options CommandOptions,
) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		cli, err := clientutil.NewClient(options.GlobalConfig)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		kinds, err := cli.ReconcileService.ListKinds(cmd.Context())
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		return kinds, cobra.ShellCompDirectiveNoFileComp
	}
}
