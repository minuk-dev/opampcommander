// Package client provides a client for the opampcommander API server.
package client

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/go-resty/resty/v2"

	apiv1 "github.com/minuk-dev/opampcommander/api/v1"
	v1version "github.com/minuk-dev/opampcommander/api/v1/version"
)

// Client is a struct that contains the endpoint and the resty client.
type Client struct {
	Endpoint string
	common   service

	AgentService        *AgentService
	AgentGroupService   *AgentGroupService
	AgentPackageService *AgentPackageService
	ConnectionService   *ConnectionService
	AuthService         *AuthService
}

type service struct {
	Resty *resty.Client
}

// New creates a new client for opampcommander's apiserver.
func New(endpoint string, opt ...Option) *Client {
	service := service{
		Resty: resty.New().SetBaseURL(endpoint),
	}
	client := &Client{
		Endpoint: endpoint,
		common:   service,

		// Initialize services to nil, they will be set later
		AgentService:        nil,
		AgentGroupService:   nil,
		AgentPackageService: nil,
		ConnectionService:   nil,
		AuthService:         nil,
	}

	for _, o := range opt {
		o.Apply(client)
	}

	client.AgentService = NewAgentService(&service)
	client.ConnectionService = NewConnectionService(&service)
	client.AuthService = NewAuthService(&service)
	client.AgentGroupService = NewAgentGroupService(&service)
	client.AgentPackageService = NewAgentPackageService(&service)

	return client
}

// GetServerVersion retrieves the server version information.
func (c *Client) GetServerVersion(ctx context.Context) (*v1version.Info, error) {
	var versionInfo v1version.Info

	res, err := c.common.Resty.R().
		SetContext(ctx).
		SetResult(&versionInfo).
		Get("/api/v1/version")
	if err != nil {
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to get version: %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	if res.Result() == nil {
		return nil, fmt.Errorf("failed to get version: %w", ErrEmptyResponse)
	}

	return &versionInfo, nil
}

// SetAuthToken sets the authentication token for the client.
func (c *Client) SetAuthToken(bearerToken string) {
	c.common.Resty.SetAuthToken(bearerToken)
}

// SetLogger sets the logger for the client.
func (c *Client) SetLogger(logger *slog.Logger) {
	c.common.Resty.SetLogger(&loggerWrapper{Logger: logger})
}

// SetVerbose enables verbose logging for the client.
func (c *Client) SetVerbose(verbose bool) {
	c.common.Resty.SetDebug(verbose)
}

func getResource[T any](ctx context.Context, c *service, url string, id string) (*T, error) {
	var result T

	res, err := c.Resty.R().
		SetContext(ctx).
		SetPathParam("id", id).
		SetResult(&result).
		Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource: %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to get resource: %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	if res.Result() == nil {
		return nil, fmt.Errorf("failed to get resource: %w", ErrEmptyResponse)
	}

	return &result, nil
}

func listResources[T any](
	ctx context.Context,
	service *service,
	url string,
	option ListSettings,
) (*apiv1.ListResponse[T], error) {
	var listResponse apiv1.ListResponse[T]

	req := service.Resty.R().
		SetContext(ctx).
		SetResult(&listResponse)

	if option.limit != nil {
		req.SetQueryParam("limit", strconv.Itoa(*option.limit))
	}

	if option.continueToken != nil {
		req.SetQueryParam("continue", *option.continueToken)
	}

	res, err := req.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to list resources(responseError): %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	if res.Result() == nil {
		return nil, fmt.Errorf("failed to list resources: %w", ErrEmptyResponse)
	}

	return &listResponse, nil
}

func createResource[Request any, Response any](
	ctx context.Context,
	service *service,
	url string,
	request *Request,
) (*Response, error) {
	var result Response

	res, err := service.Resty.R().
		SetContext(ctx).
		SetBody(request).
		SetResult(&result).
		Post(url)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to create resource(responseError): %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return &result, nil
}

func updateResource[Resource any](
	ctx context.Context,
	service *service,
	url string,
	resource *Resource,
) (*Resource, error) {
	var result Resource

	res, err := service.Resty.R().
		SetContext(ctx).
		SetBody(resource).
		SetResult(&result).
		Put(url)
	if err != nil {
		return nil, fmt.Errorf("failed to update resource(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to update resource(responseError): %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return &result, nil
}

func deleteResource(ctx context.Context, service *service, url string, name string) error {
	res, err := service.Resty.R().
		SetContext(ctx).
		SetPathParam("id", name).
		Delete(url)
	if err != nil {
		return fmt.Errorf("failed to delete resource(restyError): %w", err)
	}

	if res.IsError() {
		return fmt.Errorf("failed to delete resource(responseError): %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return nil
}
