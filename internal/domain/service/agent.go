package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

var ErrNotImplemented = errors.New("not implemented")

var _ port.AgentUsecase = (*AgentService)(nil)

type AgentService struct{}

func NewAgentService() *AgentService {
	return &AgentService{}
}

func (*AgentService) GetAgent(context.Context, uuid.UUID) (*model.Agent, error) {
	return nil, ErrNotImplemented
}

func (*AgentService) SaveAgent(context.Context, *model.Agent) error {
	return nil
}

func (*AgentService) ListAgents(context.Context) []*model.Agent {
	return nil
}
