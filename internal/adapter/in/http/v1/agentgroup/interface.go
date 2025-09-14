package agentgroup

import (
	applicationport "github.com/minuk-dev/opampcommander/internal/application/port"
)

// Usecase is an alias for the AgentGroupManageUsecase interface.
type Usecase = applicationport.AgentGroupManageUsecase

// CreateAgentGroupCommand is an alias for the applicationport.CreateAgentGroupCommand struct.
type CreateAgentGroupCommand = applicationport.CreateAgentGroupCommand
