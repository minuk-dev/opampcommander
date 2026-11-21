package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
)

// RuntimeClient checks the Commander namespace identity with a dedicated ServiceAccount token.
type RuntimeClient struct {
	endpoint *url.URL
	http     *http.Client
}

// NewRuntimeClient builds a TLS client for the configured Commander origin.
func NewRuntimeClient(address, caFile, tokenFile string) (*RuntimeClient, error) {
	endpoint, err := url.Parse(address)
	if err != nil || endpoint.Scheme != "https" || endpoint.Hostname() == "" ||
		endpoint.User != nil || endpoint.Fragment != "" || endpoint.RawQuery != "" || endpoint.ForceQuery ||
		(endpoint.Path != "" && endpoint.Path != "/") {
		return nil, errors.New("runtime URL must be an HTTPS origin")
	}
	if tokenFile == "" {
		return nil, errors.New("runtime token file is required")
	}
	httpClient, err := rest.HTTPClientFor(&rest.Config{
		TLSClientConfig: rest.TLSClientConfig{CAFile: caFile},
		BearerTokenFile: tokenFile, Timeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("create runtime client: %w", err)
	}
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &RuntimeClient{endpoint: endpoint, http: httpClient}, nil
}

func (c *RuntimeClient) verify(ctx context.Context, name string, installation, namespace types.UID) error {
	endpoint := c.endpoint.JoinPath("api/runtime/v1/namespaces", name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("runtime namespace lookup: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("runtime namespace lookup returned HTTP %d", resp.StatusCode)
	}
	var scope struct {
		Name            string    `json:"name"`
		InstallationUID types.UID `json:"installationUID"`
		NamespaceUID    types.UID `json:"namespaceUID"`
	}
	const maxSize = 4096
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxSize+1))
	if err != nil || len(data) > maxSize || json.Unmarshal(data, &scope) != nil {
		return errors.New("invalid runtime namespace response")
	}
	if scope.Name != name || namespace == "" || scope.NamespaceUID != namespace || scope.InstallationUID != installation {
		return errors.New("runtime namespace identity mismatch")
	}
	return nil
}
