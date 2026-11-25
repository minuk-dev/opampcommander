// Package kubeplugin provides kubectl opamp commands using Kubernetes credentials.
package kubeplugin

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/minuk-dev/opampcommander/pkg/version"
)

var (
	errNoManifest        = errors.New("supply a named Commander CR manifest with -f FILE or -f -")
	errExplicitNamespace = errors.New("manifest namespace differs from --namespace")
	errUnsupported       = errors.New("expected an opampcommander.io/v1alpha1 configuration resource")
)

// NewCommand creates the plugin command tree.
func NewCommand() *cobra.Command {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{}
	//exhaustruct:ignore
	root := &cobra.Command{
		Use: "opamp", Short: "Work with Commander in Kubernetes",
		Version: version.Get().GitVersion, SilenceUsage: true,
		// Also used under opampctl, where the parent normally loads legacy credentials.
		PersistentPreRunE: func(*cobra.Command, []string) error { return nil },
	}
	root.PersistentFlags().StringVar(&rules.ExplicitPath, "kubeconfig", "", "Kubeconfig file")
	root.PersistentFlags().StringVar(&overrides.CurrentContext, "context", "", "Kubernetes context")
	root.PersistentFlags().StringVarP(&overrides.Context.Namespace, "namespace", "n", "", "Kubernetes namespace")

	root.AddCommand(newRuntimeScopeCommand(rules, overrides))

	var filename string
	//exhaustruct:ignore
	validate := &cobra.Command{
		Use: "validate -f FILE", Short: "Validate Commander resources with server-side dry-run apply", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides)

			namespace, explicit, err := config.Namespace()
			if err != nil {
				return fmt.Errorf("resolve namespace: %w", err)
			}

			restConfig, err := config.ClientConfig()
			if err != nil {
				return fmt.Errorf("load kubeconfig: %w", err)
			}

			client, err := dynamic.NewForConfig(restConfig)
			if err != nil {
				return fmt.Errorf("create Kubernetes client: %w", err)
			}

			input := cmd.InOrStdin()

			if filename != "-" {
				file, err := os.Open(filepath.Clean(filename))
				if err != nil {
					return fmt.Errorf("open manifest: %w", err)
				}
				defer func() { _ = file.Close() }()

				input = file
			}

			return validateStream(cmd, client, input, namespace, explicit)
		},
	}
	validate.Flags().StringVarP(&filename, "filename", "f", "", "Manifest file or - for stdin")
	_ = validate.MarkFlagRequired("filename")
	root.AddCommand(validate)

	return root
}

func validateStream(cmd *cobra.Command, client dynamic.Interface, input io.Reader,
	namespace string, explicit bool) error {
	reader := yaml.NewYAMLReader(bufio.NewReader(input))
	count := 0

	for {
		data, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return fmt.Errorf("read manifest: %w", err)
		}

		var manifest *metav1.PartialObjectMetadata

		err = yaml.Unmarshal(data, &manifest)
		if err != nil {
			return fmt.Errorf("decode manifest: %w", err)
		}

		if manifest == nil {
			continue
		}

		err = validateManifest(cmd, client, manifest, data, namespace, explicit)
		if err != nil {
			return err
		}

		count++
	}

	if count == 0 {
		return errNoManifest
	}

	return nil
}

func validateManifest(cmd *cobra.Command, client dynamic.Interface, manifest *metav1.PartialObjectMetadata,
	data []byte, namespace string, explicit bool) error {
	target := manifest.Namespace
	if target == "" {
		target = namespace
	} else if explicit && target != namespace {
		return fmt.Errorf("%w: %s, expected %s", errExplicitNamespace, target, namespace)
	}

	name := manifest.Name
	if name == "" {
		return errNoManifest
	}

	switch manifest.Kind {
	case "AgentGroup", "RemoteConfig", "RemoteConfigSchema", "Endpoint", "AgentPackage":
	default:
		return fmt.Errorf("%w: %s", errUnsupported, manifest.Kind)
	}

	if manifest.APIVersion != "opampcommander.io/v1alpha1" {
		return fmt.Errorf("%w: %s", errUnsupported, manifest.APIVersion)
	}

	resource := schema.GroupVersionResource{
		Group: "opampcommander.io", Version: "v1alpha1", Resource: strings.ToLower(manifest.Kind) + "s",
	}
	// Preserve the original document so the server sees duplicate and unknown fields.
	//exhaustruct:ignore
	options := metav1.PatchOptions{
		DryRun: []string{metav1.DryRunAll}, FieldValidation: metav1.FieldValidationStrict, FieldManager: "kubectl",
	}

	_, err := client.Resource(resource).Namespace(target).Patch(cmd.Context(), name, types.ApplyPatchType, data, options)
	if err != nil {
		return fmt.Errorf("validate %s %s/%s: %w", manifest.Kind, target, name, err)
	}

	_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s %s/%s validated (server dry run)\n", manifest.Kind, target, name)
	if err != nil {
		return fmt.Errorf("write result: %w", err)
	}

	return nil
}
