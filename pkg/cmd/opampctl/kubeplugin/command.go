// Package kubeplugin provides kubectl opamp commands using Kubernetes credentials.
package kubeplugin

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/minuk-dev/opampcommander/pkg/version"
)

var (
	errInvalidManifest   = errors.New("invalid manifest")
	errValidationFailed  = errors.New("server dry-run request failed")
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

			// Admission warnings can quote rejected config or resolved Secret data.
			restConfig.WarningHandlerWithContext = rest.NoWarnings{}

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
			return fmt.Errorf("read document %d: %w", count+1, errInvalidManifest)
		}

		var manifest *metav1.PartialObjectMetadata

		err = yaml.Unmarshal(data, &manifest)
		if err != nil {
			return fmt.Errorf("decode document %d: %w", count+1, errInvalidManifest)
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
		return errUnsupported
	}

	if manifest.APIVersion != "opampcommander.io/v1alpha1" {
		return errUnsupported
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
		return fmt.Errorf("validate %s %s/%s: %w", manifest.Kind, target, name, safeValidationError(err))
	}

	_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s %s/%s validated (server dry run)\n", manifest.Kind, target, name)
	if err != nil {
		return fmt.Errorf("write result: %w", err)
	}

	return nil
}

// Status messages and causes can contain the full submitted value or webhook output.
func safeValidationError(err error) error {
	var status apierrors.APIStatus
	if errors.As(err, &status) {
		code := int(status.Status().Code)

		return fmt.Errorf("%w (HTTP %d %s)", errValidationFailed, code, http.StatusText(code))
	}

	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}

	return errValidationFailed
}
