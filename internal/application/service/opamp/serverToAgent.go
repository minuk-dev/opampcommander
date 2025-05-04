package opamp

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

// createServerToAgent creates a ServerToAgent message from the agent.
func (s *Service) createServerToAgent(agent *model.Agent) (*protobufs.ServerToAgent, error) {
	var flags uint64

	if agent == nil || agent.ReportFullState {
		flags |= uint64(protobufs.ServerToAgentFlags_ServerToAgentFlags_ReportFullState)
	}

	instanceUID := agent.InstanceUID

	err := agent.ResetByServerToAgent()
	if err != nil {
		return nil, fmt.Errorf("failed to reset agent: %w", err)
	}

	//exhaustruct:ignore
	return &protobufs.ServerToAgent{
		InstanceUid: instanceUID[:],
		Flags:       flags,
	}, nil
}

// createFallbackServerToAgent creates a fallback ServerToAgent message.
// This is used when the agent is not found or when there is an error in creating
// the ServerToAgent message.
func (s *Service) createFallbackServerToAgent(
	instanceUID uuid.UUID,
) *protobufs.ServerToAgent {
	//exhaustruct:ignore
	return &protobufs.ServerToAgent{
		InstanceUid: instanceUID[:],
	}
}
