// Package client provides a client for the opampcommander API server.
package client

import (
	"fmt"
	"log/slog"

	"github.com/go-resty/resty/v2"
	uuid "github.com/google/uuid"

	apiv1 "github.com/minuk-dev/opampcommander/api/v1"
)

// Client is a struct that contains the endpoint and the resty client.
type Client struct {
	Endpoint string
	common   service

	AgentService      *AgentService
	ConnectionService *ConnectionService
	AuthService       *AuthService
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
		AgentService:      nil,
		ConnectionService: nil,
		AuthService:       nil,
	}

	for _, o := range opt {
		o.Apply(client)
	}

	client.AgentService = NewAgentService(&service)
	client.ConnectionService = NewConnectionService(&service)
	client.AuthService = NewAuthService(&service)

	return client
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

// Generic function for GET requests.
func getResource[T any](c *service, url string, id uuid.UUID) (*T, error) {
	var result T

	res, err := c.Resty.R().
		SetPathParam("id", id.String()).
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

func listResources[T any](c *service, url string) (*apiv1.ListResponse[T], error) {
	var listResponse apiv1.ListResponse[T]

	res, err := c.Resty.R().
		SetResult(&listResponse).
		Get(url)
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
