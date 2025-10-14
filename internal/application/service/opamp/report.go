package opamp

import (
	"fmt"

	"github.com/open-telemetry/opamp-go/protobufs"

	"github.com/minuk-dev/opampcommander/internal/domain/model"
	agentmodel "github.com/minuk-dev/opampcommander/internal/domain/model/agent"
)

func (s *Service) report(
	agent *model.Agent,
	agentToServer *protobufs.AgentToServer,
	by *model.Server,
) error {
	// Update communication info
	agent.MarkAsCommunicated(by, s.clock.Now())

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
