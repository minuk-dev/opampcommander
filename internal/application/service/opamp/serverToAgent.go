package opamp

import (
	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
)

// createServerToAgent creates a ServerToAgent message from the agent.
func (s *Service) createServerToAgent(agent *model.Agent) (*protobufs.ServerToAgent, error) {
	var flags uint64

	// Request ReportFullState if:
	// 1. Agent has a pending ReportFullState command
	// 2. Agent's Metadata is not complete (missing description or capabilities)
	if agent.Commands.HasReportFullStateCommand() || !agent.Metadata.IsComplete() {
		flags |= uint64(protobufs.ServerToAgentFlags_ServerToAgentFlags_ReportFullState)
	}

	instanceUID := agent.Metadata.InstanceUID

	// Clear all commands after processing
	agent.Commands.Clear()

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
