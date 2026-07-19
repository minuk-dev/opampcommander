package opamp

import (
	"context"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
)

// fetchServerToAgent creates a ServerToAgent message from the agent by delegating to the
// shared domain builder. Both this OpAMP hot path and the cross-server push path use the
// same builder, so the message they produce cannot diverge.
func (s *Service) fetchServerToAgent(ctx context.Context, agentModel *agentmodel.Agent) *protobufs.ServerToAgent {
	return s.serverToAgentBuilder.Build(ctx, agentModel)
}

// createFallbackServerToAgent creates a fallback ServerToAgent message.
// This is used when the agent is not found or when there is an error in creating
// the ServerToAgent message.
func (s *Service) createFallbackServerToAgent(
	instanceUID uuid.UUID,
) *protobufs.ServerToAgent {
	return &protobufs.ServerToAgent{
		InstanceUid: instanceUID[:],
	}
}
