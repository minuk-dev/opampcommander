package kubeplugin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	coreclient "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/minuk-dev/opampcommander/pkg/formatter"
)

const (
	runtimeScopeTimeout = 10 * time.Second
	installationLabel   = "opampcommander.io/installation-uid"
)

var (
	errRuntimeNamespace = errors.New("expected a valid Kubernetes namespace name")
	errRuntimeOutput    = errors.New("--output must be text, json, or yaml")
	errRuntimeAnchor    = errors.New("expected an installation ConfigMap name and namespace")
	errRuntimeScope     = errors.New("namespace is not assigned to this live installation")
)

//nolint:tagliatelle // Match the established runtime API's UID field names.
type runtimeScope struct {
	Name            string `json:"name"            text:"NAME"`
	NamespaceUID    string `json:"namespaceUID"    text:"NAMESPACE UID"`
	InstallationUID string `json:"installationUID" text:"INSTALLATION UID"`
}

type runtimeScopeOptions struct {
	installationName      string
	installationNamespace string
	output                string
}

func newRuntimeScopeCommand(
	rules *clientcmd.ClientConfigLoadingRules, overrides *clientcmd.ConfigOverrides,
) *cobra.Command {
	var options runtimeScopeOptions
	//exhaustruct:ignore
	command := &cobra.Command{
		Use:   "runtime-scope",
		Short: "Resolve Commander namespace and installation UIDs from Kubernetes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return options.run(cmd, clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides))
		},
	}
	flags := command.Flags()
	flags.StringVar(&options.installationName, "installation-name", "", "Name of the installation ConfigMap")
	flags.StringVar(&options.installationNamespace, "installation-namespace", "", "Installation ConfigMap namespace")
	flags.StringVarP(&options.output, "output", "o", "text", "Output format (text, json, yaml)")

	_ = command.MarkFlagRequired("installation-name")
	_ = command.MarkFlagRequired("installation-namespace")

	return command
}

func (options runtimeScopeOptions) run(cmd *cobra.Command, config clientcmd.ClientConfig) error {
	format := formatter.FormatType(options.output)
	if format != formatter.TEXT && format != formatter.JSON && format != formatter.YAML {
		return errRuntimeOutput
	}

	namespace, _, err := config.Namespace()
	if err != nil {
		return fmt.Errorf("resolve namespace: %w", err)
	}

	if len(validation.IsDNS1123Label(namespace)) != 0 {
		return errRuntimeNamespace
	}

	if len(validation.IsDNS1123Label(options.installationNamespace)) != 0 ||
		len(validation.IsDNS1123Subdomain(options.installationName)) != 0 {
		return errRuntimeAnchor
	}

	restConfig, err := config.ClientConfig()
	if err != nil {
		return fmt.Errorf("load kubeconfig: %w", err)
	}

	client, err := coreclient.NewForConfig(restConfig)
	if err != nil {
		return fmt.Errorf("create Kubernetes client: %w", err)
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), runtimeScopeTimeout)
	defer cancel()

	scope, err := options.lookup(ctx, client, namespace)
	if err != nil {
		return err
	}

	err = formatter.Format(cmd.OutOrStdout(), scope, format)
	if err != nil {
		return fmt.Errorf("write runtime scope: %w", err)
	}

	return nil
}

func (options runtimeScopeOptions) lookup(
	ctx context.Context, client coreclient.CoreV1Interface, namespaceName string,
) (runtimeScope, error) {
	namespace, err := client.Namespaces().Get(ctx, namespaceName, metav1.GetOptions{})
	if err != nil {
		return runtimeScope{}, fmt.Errorf("get namespace %s: %w", namespaceName, err)
	}

	anchor, err := client.ConfigMaps(options.installationNamespace).Get(ctx, options.installationName, metav1.GetOptions{})
	if err != nil {
		return runtimeScope{}, fmt.Errorf("get installation ConfigMap: %w", err)
	}

	if namespace.UID == "" || namespace.DeletionTimestamp != nil || anchor.UID == "" || anchor.DeletionTimestamp != nil ||
		namespace.Labels[installationLabel] != string(anchor.UID) {
		return runtimeScope{}, errRuntimeScope
	}

	return runtimeScope{
		Name: namespace.Name, NamespaceUID: string(namespace.UID), InstallationUID: string(anchor.UID),
	}, nil
}
