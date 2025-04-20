// Package client provides a client for the opampcommander API server.
package client

import (
	"fmt"

	"github.com/go-resty/resty/v2"
	uuid "github.com/google/uuid"
)

// Client is a struct that contains the endpoint and the resty client.
type Client struct {
	Endpoint string
	Client   *resty.Client
}

// NewClient creates a new client for opampcommander's apiserver.
func NewClient(endpoint string) *Client {
	return &Client{
		Endpoint: endpoint,
		Client:   resty.New().SetBaseURL(endpoint),
	}
}

// Generic function for GET requests.
func getResource[T any](c *Client, url string, id uuid.UUID) (*T, error) {
	var result T

	res, err := c.Client.R().
		SetPathParam("id", id.String()).
		SetResult(&result).
		Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource: %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to get resource: %w", &ResponseError{
			StatusCode: res.StatusCode(),
		})
	}

	if res.Result() == nil {
		return nil, fmt.Errorf("failed to get resource: %w", ErrEmptyResponse)
	}

	return &result, nil
}

func listResources[T any](c *Client, url string) ([]T, error) {
	var result []T

	res, err := c.Client.R().
		SetResult(&result).
		Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to list resources(responseError): %w", &ResponseError{
			StatusCode: res.StatusCode(),
		})
	}

	if res.Result() == nil {
		return nil, fmt.Errorf("failed to list resources: %w", ErrEmptyResponse)
	}

	return result, nil
}
