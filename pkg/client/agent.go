package client

import (
	uuid "github.com/google/uuid"

	agentv1 "github.com/minuk-dev/opampcommander/api/v1/agent"
)

const (
	ListAgentURL = "/v1/agents"
	GetAgentURL  = "/v1/agents/:id"
)

func (c *Client) GetAgent(id uuid.UUID) (*agentv1.Agent, error) {
	return getResource[agentv1.Agent](c, GetAgentURL, id)
}

func (c *Client) ListAgents() ([]*agentv1.Agent, error) {
	return listResources[*agentv1.Agent](c, ListAgentURL)
}
