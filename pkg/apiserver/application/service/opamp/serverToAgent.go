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

// createErrorServerToAgent builds a ServerToAgent that carries only an error_response, giving
// the agent a structured signal that the server could not process its message. Per the OpAMP
// spec the agent ignores every other ServerToAgent field when error_response is set, so no
// desired-state fields are populated here — sending them would be wasted work the agent discards.
func (s *Service) createErrorServerToAgent(
	instanceUID uuid.UUID,
	errorType protobufs.ServerErrorResponseType,
	message string,
) *protobufs.ServerToAgent {
	return &protobufs.ServerToAgent{
		InstanceUid: instanceUID[:],
		ErrorResponse: &protobufs.ServerErrorResponse{
			Type:         errorType,
			ErrorMessage: message,
		},
	}
}
