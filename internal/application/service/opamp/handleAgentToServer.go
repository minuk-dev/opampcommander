package opamp

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/open-telemetry/opamp-go/protobufs"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/model/agent"
)

// HandleAgentToServer handle a message from agent.
func (s *Service) HandleAgentToServer(ctx context.Context, agentToServer *protobufs.AgentToServer) error {
	instanceUID := uuid.UUID(agentToServer.GetInstanceUid())
	s.logger.Info("HandleAgentToServer",
		slog.String("instanceUID", instanceUID.String()),
		slog.String("message", "start"),
	)
	s.logger.Debug("HandleAgentToServer",
		slog.String("instanceUID", instanceUID.String()),
		slog.String("message", "agentToServer"),
		slog.Any("agentToServer", agentToServer),
	)

	agent, err := s.agentUsecase.GetOrCreateAgent(ctx, instanceUID)
	if err != nil {
		return fmt.Errorf("failed to get or create agent: %w", err)
	}

	s.logger.Debug("HandleAgentToServer",
		slog.String("instanceUID", instanceUID.String()),
		slog.String("message", "get or create agent"),
		slog.Any("agent", agent),
	)

	err = s.report(agent, agentToServer)
	if err != nil {
		return fmt.Errorf("failed to report: %w", err)
	}

	s.logger.Debug("HandleAgentToServer",
		slog.String("instanceUID", instanceUID.String()),
		slog.String("message", "report"),
		slog.Any("agent", agent),
	)

	if !agent.IsManaged() {
		agent.SetReportFullState(true)
	}

	err = s.agentUsecase.SaveAgent(ctx, agent)
	if err != nil {
		return fmt.Errorf("failed to save agent: %w", err)
	}

	s.logger.Info("HandleAgentToServer",
		slog.String("instanceUID", instanceUID.String()),
		slog.String("message", "success"),
	)

	_, err = s.connectionUsecase.GetOrCreateConnection(instanceUID)
	if err != nil {
		return fmt.Errorf("failed to get or create connection: %w", err)
	}

	return nil
}

func (s *Service) report(agent *model.Agent, agentToServer *protobufs.AgentToServer) error {
	err := agent.ReportDescription(descToDomain(agentToServer.GetAgentDescription()))
	if err != nil {
		return fmt.Errorf("failed to report description: %w", err)
	}

	err = agent.ReportComponentHealth(healthToDomain(agentToServer.GetHealth()))
	if err != nil {
		return fmt.Errorf("failed to report component health: %w", err)
	}

	capabilities := agentToServer.GetCapabilities()

	err = agent.ReportCapabilities((*agentmodel.Capabilities)(&capabilities))
	if err != nil {
		return fmt.Errorf("failed to report capabilities: %w", err)
	}

	err = agent.ReportEffectiveConfig(effectiveConfigToDomain(agentToServer.GetEffectiveConfig()))
	if err != nil {
		return fmt.Errorf("failed to report effective config: %w", err)
	}

	err = agent.ReportRemoteConfigStatus(remoteConfigStatusToDomain(agentToServer.GetRemoteConfigStatus()))
	if err != nil {
		return fmt.Errorf("failed to report remote config status: %w", err)
	}

	err = agent.ReportPackageStatuses(packageStatusToDomain(agentToServer.GetPackageStatuses()))
	if err != nil {
		return fmt.Errorf("failed to report package statuses: %w", err)
	}

	err = agent.ReportCustomCapabilities(customCapabilitiesToDomain(agentToServer.GetCustomCapabilities()))
	if err != nil {
		return fmt.Errorf("failed to report custom capabilities: %w", err)
	}

	err = agent.ReportAvailableComponents(availableComponentsToDomain(agentToServer.GetAvailableComponents()))
	if err != nil {
		return fmt.Errorf("failed to report available components: %w", err)
	}

	return nil
}
