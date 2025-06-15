package client

import (
	uuid "github.com/google/uuid"

	agentv1 "github.com/minuk-dev/opampcommander/api/v1/agent"
)

const (
	// ListAgentURL is the path to list all agents.
	ListAgentURL = "/api/v1/agents"
	// GetAgentURL is the path to get an agent by ID.
	GetAgentURL = "/api/v1/agents/:id"
)

// AgentService provides methods to interact with agents.
type AgentService struct {
	service *service
}

// NewAgentService creates a new AgentService.
func NewAgentService(service *service) *AgentService {
	return &AgentService{
		service: service,
	}
}

// GetAgent retrieves an agent by its ID.
func (s *AgentService) GetAgent(id uuid.UUID) (*agentv1.Agent, error) {
	return getResource[agentv1.Agent](s.service, GetAgentURL, id)
}

// ListAgents lists all agents.
func (s *AgentService) ListAgents() ([]*agentv1.Agent, error) {
	return listResources[*agentv1.Agent](s.service, ListAgentURL)
}
