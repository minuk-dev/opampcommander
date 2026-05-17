// Package namespace provides the command to get namespace information.
package namespace

import (
	"errors"
	"fmt"

	"github.com/samber/lo"
	"github.com/spf13/cobra"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

// ErrCommandExecutionFailed is returned when the command execution fails.
var ErrCommandExecutionFailed = errors.New("command execution failed")

// CommandOptions contains the options for the namespace command.
type CommandOptions struct {
	*config.GlobalConfig

	formatType     string
	includeDeleted bool
	client         *client.Client
}

// NewCommand creates a new namespace command.
func NewCommand(options CommandOptions) *cobra.Command {
	//exhaustruct:ignore
	cmd := &cobra.Command{
		Use:   "namespace",
		Short: "namespace",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := options.Prepare(cmd, args)
			if err != nil {
				return err
			}

			return options.Run(cmd, args)
		},
	}
	cmd.Flags().StringVarP(
		&options.formatType, "output", "o", "short",
		"Output format (short, text, json, yaml)",
	)
	cmd.Flags().BoolVar(
		&options.includeDeleted, "include-deleted", false,
		"Include soft-deleted namespaces",
	)

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

// ItemForCLI is a struct for namespace display.
type ItemForCLI struct {
	Name      string `short:"Name"       text:"Name"`
	CreatedAt string `short:"Created At" text:"Created At"`
}

// Run runs the command.
func (opt *CommandOptions) Run(
	cmd *cobra.Command, args []string,
) error {
	if len(args) > 0 {
		return opt.Get(cmd, args)
	}

	return opt.List(cmd)
}

// List retrieves the list of namespaces.
func (opt *CommandOptions) List(cmd *cobra.Command) error {
	listOpts := []client.ListOption{client.WithIncludeDeleted(opt.includeDeleted)}

	resp, err := opt.client.NamespaceService.ListNamespaces(
		cmd.Context(), listOpts...,
	)
	if err != nil {
		return fmt.Errorf("failed to list namespaces: %w", err)
	}

	formatType := formatter.FormatType(opt.formatType)

	switch formatType {
	case formatter.SHORT, formatter.TEXT:
		items := lo.Map(
			resp.Items,
			func(item v1.Namespace, _ int) ItemForCLI {
				return ItemForCLI{
					Name:      item.Metadata.Name,
					CreatedAt: item.Metadata.CreatedAt.String(),
				}
			},
		)

		err = formatter.Format(
			cmd.OutOrStdout(), items, formatType,
		)
	case formatter.JSON, formatter.YAML:
		err = formatter.Format(
			cmd.OutOrStdout(), resp.Items, formatType,
		)
	default:
		return fmt.Errorf(
			"unsupported format type: %s, %w",
			opt.formatType, ErrCommandExecutionFailed,
		)
	}

	if err != nil {
		return fmt.Errorf("failed to format namespaces: %w", err)
	}

	return nil
}

// Get retrieves namespace(s) by name.
func (opt *CommandOptions) Get( //nolint:funlen // CLI display logic requires branching
	cmd *cobra.Command, names []string,
) error {
	getOpts := []client.GetOption{client.WithGetIncludeDeleted(opt.includeDeleted)}

	namespaces := make([]v1.Namespace, 0, len(names))

	for _, name := range names {
		result, err := opt.client.NamespaceService.GetNamespace(
			cmd.Context(), name, getOpts...,
		)
		if err != nil {
			cmd.PrintErrf(
				"failed to get namespace %s: %v\n", name, err,
			)

			continue
		}

		namespaces = append(namespaces, *result)
	}

	if len(namespaces) == 0 {
		cmd.Println("No namespaces found.")

		return nil
	}

	formatType := formatter.FormatType(opt.formatType)

	switch formatType {
	case formatter.SHORT, formatter.TEXT:
		items := lo.Map(
			namespaces,
			func(item v1.Namespace, _ int) ItemForCLI {
				return ItemForCLI{
					Name:      item.Metadata.Name,
					CreatedAt: item.Metadata.CreatedAt.String(),
				}
			},
		)

		err := formatter.Format(
			cmd.OutOrStdout(), items, formatType,
		)
		if err != nil {
			return fmt.Errorf("failed to format: %w", err)
		}

		return nil
	case formatter.JSON, formatter.YAML:
		err := formatter.Format(
			cmd.OutOrStdout(), namespaces, formatType,
		)
		if err != nil {
			return fmt.Errorf("failed to format: %w", err)
		}

		return nil
	default:
		return fmt.Errorf(
			"unsupported format type: %s, %w",
			opt.formatType, ErrCommandExecutionFailed,
		)
	}
}
