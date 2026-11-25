package kubeplugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/minuk-dev/opampcommander/pkg/formatter"
)

const (
	runtimeScopeTimeout = 10 * time.Second
	maxRuntimeScopeBody = 4096
)

var (
	errRuntimeURL       = errors.New("--runtime-url must be an HTTPS origin")
	errRuntimeTokenFile = errors.New("--runtime-token-file must name a file containing a runtime audience token")
	errRuntimeNamespace = errors.New("expected a valid Kubernetes namespace name")
	errRuntimeOutput    = errors.New("--output must be text, json, or yaml")
	errRuntimeStatus    = errors.New("runtime namespace lookup failed")
	errRuntimeScope     = errors.New("runtime returned an invalid namespace scope")
)

//nolint:tagliatelle // Match the established runtime API's UID field names.
type runtimeScope struct {
	Name            string `json:"name"            text:"NAME"`
	NamespaceUID    string `json:"namespaceUID"    text:"NAMESPACE UID"`
	InstallationUID string `json:"installationUID" text:"INSTALLATION UID"`
}

type runtimeScopeOptions struct {
	origin    string
	caFile    string
	tokenFile string
	output    string
}

func newRuntimeScopeCommand(
	rules *clientcmd.ClientConfigLoadingRules, overrides *clientcmd.ConfigOverrides,
) *cobra.Command {
	var options runtimeScopeOptions
	//exhaustruct:ignore
	command := &cobra.Command{
		Use:   "runtime-scope",
		Short: "Resolve the selected namespace and installation UIDs from Commander",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return options.run(cmd, clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides))
		},
	}
	flags := command.Flags()
	flags.StringVar(&options.origin, "runtime-url", "", "Trusted HTTPS origin of the Commander runtime API")
	flags.StringVar(&options.caFile, "runtime-ca-file", "", "PEM CA bundle for Commander (default: system roots)")
	flags.StringVar(&options.tokenFile, "runtime-token-file", "", "File with an audience-scoped runtime bearer token")
	flags.StringVarP(&options.output, "output", "o", "text", "Output format (text, json, yaml)")

	_ = command.MarkFlagRequired("runtime-url")
	_ = command.MarkFlagRequired("runtime-token-file")

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

	scope, err := options.lookup(cmd.Context(), namespace)
	if err != nil {
		return err
	}

	err = formatter.Format(cmd.OutOrStdout(), scope, format)
	if err != nil {
		return fmt.Errorf("write runtime scope: %w", err)
	}

	return nil
}

func (options runtimeScopeOptions) lookup(ctx context.Context, namespace string) (runtimeScope, error) {
	origin, err := url.Parse(options.origin)
	if err != nil || origin.Scheme != "https" || origin.Hostname() == "" || origin.User != nil ||
		(origin.EscapedPath() != "" && origin.EscapedPath() != "/") || origin.ForceQuery || origin.RawQuery != "" ||
		strings.Contains(options.origin, "#") {
		return runtimeScope{}, errRuntimeURL
	}

	if options.tokenFile == "" {
		return runtimeScope{}, errRuntimeTokenFile
	}

	// Use a separate transport so Kubernetes API credentials stay with Kubernetes.
	//exhaustruct:ignore
	client, err := rest.HTTPClientFor(&rest.Config{
		BearerTokenFile: options.tokenFile,
		//exhaustruct:ignore
		TLSClientConfig: rest.TLSClientConfig{CAFile: options.caFile},
		Timeout:         runtimeScopeTimeout,
	})
	if err != nil {
		return runtimeScope{}, fmt.Errorf("configure runtime TLS and credentials: %w", err)
	}

	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	endpoint := origin.JoinPath("api", "runtime", "v1", "namespaces", namespace)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return runtimeScope{}, fmt.Errorf("create runtime request: %w", err)
	}

	request.Header.Set("Accept", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return runtimeScope{}, fmt.Errorf("request runtime scope: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return runtimeScope{}, fmt.Errorf("%w (HTTP %d %s)", errRuntimeStatus,
			response.StatusCode, http.StatusText(response.StatusCode))
	}

	return decodeRuntimeScope(response.Body, namespace)
}

func decodeRuntimeScope(body io.Reader, namespace string) (runtimeScope, error) {
	data, err := io.ReadAll(io.LimitReader(body, maxRuntimeScopeBody+1))
	if err != nil {
		return runtimeScope{}, fmt.Errorf("read runtime scope: %w", err)
	}

	var scope runtimeScope
	if len(data) > maxRuntimeScopeBody || json.Unmarshal(data, &scope) != nil || scope.Name != namespace ||
		scope.NamespaceUID == "" || scope.InstallationUID == "" ||
		strings.ContainsFunc(scope.NamespaceUID, unicode.IsControl) ||
		strings.ContainsFunc(scope.InstallationUID, unicode.IsControl) {
		return runtimeScope{}, errRuntimeScope
	}

	return scope, nil
}
