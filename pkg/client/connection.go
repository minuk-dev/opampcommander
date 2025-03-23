package client

import (
	"fmt"

	uuid "github.com/google/uuid"

	connectionv1 "github.com/minuk-dev/opampcommander/api/v1/connection"
)

const (
	ListConnectionsPath = "/v1/connections"
	GetConnectionPath   = "/v1/connections/:id"
)

func (c *Client) GetConnection(id uuid.UUID) (*connectionv1.Connection, error) {
	res, err := c.Client.R().
		SetPathParam("id", id.String()).
		SetResult(
			//exhaustruct:ignore
			&connectionv1.Connection{},
		).
		Get(GetConnectionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to get connection: %w", &ResponseError{
			StatusCode: res.StatusCode(),
		})
	}

	if res.Result() == nil {
		return nil, fmt.Errorf("failed to get connection: %w", ErrEmptyResponse)
	}

	result, ok := res.Result().(*connectionv1.Connection)
	if !ok {
		return nil, fmt.Errorf("failed to get connection: %w", ErrUnexpectedBehavior)
	}

	return result, nil
}

func (c *Client) ListConnections() ([]*connectionv1.Connection, error) {
	res, err := c.Client.R().
		SetResult([]*connectionv1.Connection{}).
		Get(ListConnectionsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list connections(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to list connections(responseError): %w", &ResponseError{
			StatusCode: res.StatusCode(),
		})
	}

	if res.Result() == nil {
		return nil, fmt.Errorf("failed to list connections(restyResultError): %w", ErrEmptyResponse)
	}

	result, ok := res.Result().(*[]*connectionv1.Connection)
	if !ok {
		return nil, fmt.Errorf("failed to list connections: %w", ErrUnexpectedBehavior)
	}

	return *result, nil
}
