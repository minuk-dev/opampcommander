// Package patch implements partial resource updates without a preliminary GET.
package patch

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/clientutil"
	"github.com/minuk-dev/opampcommander/pkg/formatter"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

// ErrInvalidPatch is returned for missing or non-object patch input.
var ErrInvalidPatch = errors.New("provide a JSON object with --patch or --file")

const namespaceResource = "namespace"

// NewCommand creates the patch command for the four declarative resources.
func NewCommand(globalConfig *config.GlobalConfig) *cobra.Command {
	cmd := &cobra.Command{Use: "patch", Short: "Partially update an existing resource"}
	for _, resource := range []string{"agentgroup", "agentpackage", "agentremoteconfig", namespaceResource} {
		cmd.AddCommand(resourceCommand(globalConfig, resource))
	}

	return cmd
}

func resourceCommand(globalConfig *config.GlobalConfig, resource string) *cobra.Command {
	var namespace, patch, file, output string

	cmd := &cobra.Command{
		Use: resource + " NAME", Aliases: []string{resource + "s"}, Args: cobra.ExactArgs(1),
		Short: "Patch a " + resource,
		Long: "Apply JSON Merge Patch. Include metadata.resourceVersion to make the edit conditional. " +
			"Without it, later writes can overwrite the same field. Null removes nullable fields; arrays are replaced.",
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := readPatch(cmd.InOrStdin(), patch, file)
			if err != nil {
				return err
			}

			cli, err := clientutil.NewClient(globalConfig)
			if err != nil {
				return fmt.Errorf("create client: %w", err)
			}

			return run(cmd, cli, resource, namespace, args[0], data, formatter.FormatType(output))
		},
	}
	cmd.Flags().StringVarP(&patch, "patch", "p", "", "JSON merge patch (optional metadata.resourceVersion precondition)")
	cmd.Flags().StringVarP(&file, "file", "f", "", "JSON merge patch file; - reads stdin")
	cmd.MarkFlagsMutuallyExclusive("patch", "file")
	cmd.MarkFlagsOneRequired("patch", "file")
	cmd.Flags().StringVarP(&output, "output", "o", "json", "Output format (json, yaml)")

	if resource != namespaceResource {
		cmd.Flags().StringVarP(&namespace, namespaceResource, "n", "default", "Resource namespace")
	}

	return cmd
}

func readPatch(input io.Reader, patch, file string) ([]byte, error) {
	data := []byte(patch)

	var err error

	switch file {
	case "":
	case "-":
		data, err = io.ReadAll(input)
	default:
		data, err = os.ReadFile(file) //nolint:gosec // The CLI intentionally reads the user-selected file.
	}

	if err != nil {
		return nil, fmt.Errorf("read patch: %w", err)
	}

	var object map[string]json.RawMessage

	err = json.Unmarshal(data, &object)
	if err != nil || object == nil {
		return nil, ErrInvalidPatch
	}

	return data, nil
}

func run(cmd *cobra.Command, cli *client.Client, resource, namespace, name string,
	data []byte, output formatter.FormatType,
) error {
	var (
		result any
		err    error
	)

	switch resource {
	case "agentgroup":
		result, err = cli.AgentGroupService.PatchAgentGroup(cmd.Context(), namespace, name, data)
	case "agentpackage":
		result, err = cli.AgentPackageService.PatchAgentPackage(cmd.Context(), namespace, name, data)
	case "agentremoteconfig":
		result, err = cli.AgentRemoteConfigService.PatchAgentRemoteConfig(cmd.Context(), namespace, name, data)
	case namespaceResource:
		result, err = cli.NamespaceService.PatchNamespace(cmd.Context(), name, data)
	}

	if err != nil {
		return fmt.Errorf("patch %s: %w", resource, err)
	}

	err = formatter.Format(cmd.OutOrStdout(), result, output)
	if err != nil {
		return fmt.Errorf("format resource: %w", err)
	}

	return nil
}
