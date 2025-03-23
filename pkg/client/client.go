package client

import (
	"fmt"

	"github.com/go-resty/resty/v2"
	uuid "github.com/google/uuid"

	agentv1 "github.com/minuk-dev/opampcommander/api/v1/agent"
)

const (
	ListAgentURL = "/v1/agents"
	GetAgentURL  = "/v1/agents/:id"
)

type Client struct {
	Endpoint string
	Client   *resty.Client
}

func NewClient(endpoint string) *Client {
	return &Client{
		Endpoint: endpoint,
		Client:   resty.New().SetBaseURL(endpoint),
	}
}

func (c *Client) GetAgent(id uuid.UUID) (*agentv1.Agent, error) {
	res, err := c.Client.R().
		SetPathParam("id", id.String()).
		SetResult(
			//exhaustruct:ignore
			&agentv1.Agent{},
		).
		Get(GetAgentURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to get agent: %w", &ResponseError{
			StatusCode: res.StatusCode(),
		})
	}

	if res.Result() == nil {
		return nil, fmt.Errorf("failed to get agent: %w", ErrEmptyResponse)
	}

	result, ok := res.Result().(*agentv1.Agent)
	if !ok {
		return nil, fmt.Errorf("failed to get agent: %w", ErrUnexpectedBehavior)
	}

	return result, nil
}

func (c *Client) ListAgents() ([]*agentv1.Agent, error) {
	res, err := c.Client.R().
		SetResult([]*agentv1.Agent{}).
		Get(ListAgentURL)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents(restyError): %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to list agents(responseError): %w", &ResponseError{
			StatusCode: res.StatusCode(),
		})
	}

	if res.Result() == nil {
		return nil, fmt.Errorf("failed to list agents(restyResultError): %w", ErrEmptyResponse)
	}

	result, ok := res.Result().(*[]*agentv1.Agent)
	if !ok {
		return nil, fmt.Errorf("failed to list agents(type cast): %w", ErrUnexpectedBehavior)
	}

	return *result, nil
}
