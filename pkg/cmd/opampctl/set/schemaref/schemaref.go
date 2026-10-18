// Package schemaref manages a remote config's schema references.
package schemaref

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

var errInvalidReferences = errors.New("use nonempty --schema-ref names or --clear")

// CommandOptions configures the schema-reference update command.
type CommandOptions struct {
	*config.GlobalConfig

	namespace  string
	schemaRefs []string
	clear      bool
	formatType string
	client     *client.Client
}

// NewCommand creates a command that replaces or clears schema references.
func NewCommand(options CommandOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schemaref NAME (--schema-ref SCHEMA ... | --clear)",
		Short: "Replace or clear an AgentRemoteConfig's schema references",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := options.validate()
			if err != nil {
				return err
			}

			cli, err := clientutil.NewClient(options.GlobalConfig)
			if err != nil {
				return fmt.Errorf("create authenticated client: %w", err)
			}

			options.client = cli

			return options.Run(cmd, args)
		},
	}
	cmd.Flags().StringVarP(&options.namespace, "namespace", "n", "default", "Namespace")
	cmd.Flags().StringArrayVar(&options.schemaRefs, "schema-ref", nil,
		"Schema name (repeatable; replaces the existing list)")
	cmd.Flags().BoolVar(&options.clear, "clear", false, "Remove all schema references")
	cmd.Flags().StringVarP(&options.formatType, "output", "o", "yaml", "Output format (yaml, json)")
	cmd.MarkFlagsMutuallyExclusive("schema-ref", "clear")
	cmd.MarkFlagsOneRequired("schema-ref", "clear")

	return cmd
}

// Run sends a version-conditional merge patch, leaving config content and attributes untouched.
func (opts *CommandOptions) Run(cmd *cobra.Command, args []string) error {
	err := opts.validate()
	if err != nil {
		return err
	}

	resource, err := opts.client.AgentRemoteConfigService.GetAgentRemoteConfig(cmd.Context(), opts.namespace, args[0])
	if err != nil {
		return fmt.Errorf("get remote config: %w", err)
	}

	refs := opts.schemaRefs
	if opts.clear {
		refs = []string{}
	}

	patch, err := json.Marshal(map[string]any{
		"metadata": map[string]string{"resourceVersion": strconv.FormatInt(resource.Metadata.ResourceVersion, 10)},
		"spec":     map[string]any{"schemaRefs": refs},
	})
	if err != nil {
		return fmt.Errorf("encode schema reference patch: %w", err)
	}

	updated, err := opts.client.AgentRemoteConfigService.PatchAgentRemoteConfig(
		cmd.Context(), opts.namespace, args[0], patch,
	)
	if err != nil {
		return fmt.Errorf("patch schema references: %w", err)
	}

	err = formatter.Format(cmd.OutOrStdout(), updated, formatter.FormatType(opts.formatType))
	if err != nil {
		return fmt.Errorf("format remote config: %w", err)
	}

	return nil
}

func (opts *CommandOptions) validate() error {
	if (opts.clear && len(opts.schemaRefs) > 0) || (!opts.clear && len(opts.schemaRefs) == 0) {
		return errInvalidReferences
	}

	for _, ref := range opts.schemaRefs {
		if strings.TrimSpace(ref) == "" {
			return errInvalidReferences
		}
	}

	return nil
}
