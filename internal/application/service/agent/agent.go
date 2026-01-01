// Package agent provides application services for the agent
package agent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"k8s.io/utils/clock"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	v1agent "github.com/minuk-dev/opampcommander/api/v1/agent"
	"github.com/minuk-dev/opampcommander/internal/application/mapper"
	applicationport "github.com/minuk-dev/opampcommander/internal/application/port"
	"github.com/minuk-dev/opampcommander/internal/domain/model"
	"github.com/minuk-dev/opampcommander/internal/domain/model/agent"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
)

var (
	// ErrRestartCapabilityNotSupported is returned when agent doesn't support restart capability.
	ErrRestartCapabilityNotSupported = errors.New("agent does not support restart capability")
)

var _ applicationport.AgentManageUsecase = (*Service)(nil)

// Service is a struct that implements the AgentManageUsecase interface.
type Service struct {
	// domain usecases
	agentUsecase             domainport.AgentUsecase
	agentNotificationUsecase domainport.AgentNotificationUsecase

	// mapper
	mapper *mapper.Mapper
	logger *slog.Logger
	clock  clock.Clock
}

// New creates a new instance of the Service struct.
func New(
	agentUsecase domainport.AgentUsecase,
	agentNotificationUsecase domainport.AgentNotificationUsecase,
	logger *slog.Logger,
) *Service {
	return &Service{
		agentUsecase:             agentUsecase,
		agentNotificationUsecase: agentNotificationUsecase,

		mapper: mapper.New(),
		logger: logger,
		clock:  clock.RealClock{},
	}
}

// GetAgent implements port.AgentManageUsecase.
func (s *Service) GetAgent(ctx context.Context, instanceUID uuid.UUID) (*v1agent.Agent, error) {
	agent, err := s.agentUsecase.GetAgent(ctx, instanceUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	return s.mapper.MapAgentToAPI(agent), nil
}

// ListAgents implements port.AgentManageUsecase.
func (s *Service) ListAgents(
	ctx context.Context,
	options *model.ListOptions,
) (*v1agent.ListResponse, error) {
	response, err := s.agentUsecase.ListAgents(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	return v1agent.NewListResponse(
		lo.Map(response.Items, func(agent *model.Agent, _ int) v1agent.Agent {
			return *s.mapper.MapAgentToAPI(agent)
		}),
		v1.ListMeta{
			Continue:           response.Continue,
			RemainingItemCount: response.RemainingItemCount,
		},
	), nil
}

// SearchAgents implements port.AgentManageUsecase.
func (s *Service) SearchAgents(
	ctx context.Context,
	query string,
	options *model.ListOptions,
) (*v1agent.ListResponse, error) {
	response, err := s.agentUsecase.SearchAgents(ctx, query, options)
	if err != nil {
		return nil, fmt.Errorf("failed to search agents: %w", err)
	}

	return v1agent.NewListResponse(
		lo.Map(response.Items, func(agent *model.Agent, _ int) v1agent.Agent {
			return *s.mapper.MapAgentToAPI(agent)
		}),
		v1.ListMeta{
			Continue:           response.Continue,
			RemainingItemCount: response.RemainingItemCount,
		},
	), nil
}

// SetNewInstanceUID implements port.AgentManageUsecase.
func (s *Service) SetNewInstanceUID(
	ctx context.Context,
	instanceUID uuid.UUID,
	newInstanceUID uuid.UUID,
) (*v1agent.Agent, error) {
	agent, err := s.agentUsecase.GetAgent(ctx, instanceUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	// Set the new instance UID
	agent.Spec.NewInstanceUID = newInstanceUID

	// Save the updated agent
	err = s.agentUsecase.SaveAgent(ctx, agent)
	if err != nil {
		return nil, fmt.Errorf("failed to save agent: %w", err)
	}

	return s.mapper.MapAgentToAPI(agent), nil
}

// RestartAgent implements port.AgentManageUsecase.
func (s *Service) RestartAgent(ctx context.Context, instanceUID uuid.UUID) error {
	agentModel, err := s.agentUsecase.GetAgent(ctx, instanceUID)
	if err != nil {
		return fmt.Errorf("failed to get agent: %w", err)
	}

	// Check if agent supports restart capability
	if !agentModel.Metadata.Capabilities.Has(agent.AgentCapabilityAcceptsRestartCommand) {
		return fmt.Errorf("agent %s: %w", instanceUID, ErrRestartCapabilityNotSupported)
	}

	// Set the required restart time to now to trigger restart on next OpAMP message
	err = agentModel.SetRestartRequired(s.clock.Now())
	if err != nil {
		return fmt.Errorf("failed to set restart required: %w", err)
	}

	// Save the updated agent
	err = s.agentUsecase.SaveAgent(ctx, agentModel)
	if err != nil {
		return fmt.Errorf("failed to save agent with restart flag: %w", err)
	}

	// Notify the connected server that the agent needs to be restarted
	err = s.agentNotificationUsecase.NotifyAgentUpdated(ctx, agentModel)
	if err != nil {
		s.logger.Warn("failed to notify agent update for restart",
			slog.String("agentInstanceUID", instanceUID.String()),
			slog.String("error", err.Error()))
		// Don't return error as the restart flag is already set
	}

	s.logger.Info("restart scheduled for agent",
		"instanceUID", instanceUID.String(),
		"requiredRestartedAt", agentModel.Spec.RestartInfo.RequiredRestartedAt)

	return nil
}

// buildConnectionOptions builds connection options from API certificate.
func buildConnectionOptions(
	headers map[string][]string,
	cert v1agent.TLSCertificate,
) []model.ConnectionOption {
	opts := []model.ConnectionOption{}

	if len(headers) > 0 {
		opts = append(opts, model.WithHeaders(headers))
	}

	if cert.Cert != "" || cert.PrivateKey != "" || cert.CaCert != "" {
		opts = append(opts, model.WithCertificate(model.TelemetryTLSCertificate{
			Cert:       []byte(cert.Cert),
			PrivateKey: []byte(cert.PrivateKey),
			CaCert:     []byte(cert.CaCert),
		}))
	}

	return opts
}

// SetConnectionSettings sets connection settings for an agent.
//
//nolint:funlen
func (s *Service) SetConnectionSettings(
	ctx context.Context,
	instanceUID uuid.UUID,
	apiConnectionSettings v1agent.ConnectionSettings,
	requestedBy string,
) error {
	agentModel, err := s.agentUsecase.GetAgent(ctx, instanceUID)
	if err != nil {
		return fmt.Errorf("failed to get agent: %w", err)
	}

	// Set OpAMP connection settings if provided
	if apiConnectionSettings.OpAMP.DestinationEndpoint != "" {
		opampOpts := buildConnectionOptions(
			apiConnectionSettings.OpAMP.Headers,
			apiConnectionSettings.OpAMP.Certificate,
		)

		err = agentModel.SetOpAMPConnectionSettings(apiConnectionSettings.OpAMP.DestinationEndpoint, opampOpts...)
		if err != nil {
			return fmt.Errorf("failed to set OpAMP connection settings: %w", err)
		}
	}

	// Set own metrics connection settings if provided
	if apiConnectionSettings.OwnMetrics.DestinationEndpoint != "" {
		metricsOpts := buildConnectionOptions(
			apiConnectionSettings.OwnMetrics.Headers,
			apiConnectionSettings.OwnMetrics.Certificate,
		)

		err = agentModel.SetMetricsConnectionSettings(apiConnectionSettings.OwnMetrics.DestinationEndpoint, metricsOpts...)
		if err != nil {
			return fmt.Errorf("failed to set metrics connection settings: %w", err)
		}
	}

	// Set own logs connection settings if provided
	if apiConnectionSettings.OwnLogs.DestinationEndpoint != "" {
		logsOpts := buildConnectionOptions(
			apiConnectionSettings.OwnLogs.Headers,
			apiConnectionSettings.OwnLogs.Certificate,
		)

		err = agentModel.SetLogsConnectionSettings(apiConnectionSettings.OwnLogs.DestinationEndpoint, logsOpts...)
		if err != nil {
			return fmt.Errorf("failed to set logs connection settings: %w", err)
		}
	}

	// Set own traces connection settings if provided
	if apiConnectionSettings.OwnTraces.DestinationEndpoint != "" {
		tracesOpts := buildConnectionOptions(
			apiConnectionSettings.OwnTraces.Headers,
			apiConnectionSettings.OwnTraces.Certificate,
		)

		err = agentModel.SetTracesConnectionSettings(apiConnectionSettings.OwnTraces.DestinationEndpoint, tracesOpts...)
		if err != nil {
			return fmt.Errorf("failed to set traces connection settings: %w", err)
		}
	}

	// Set other connection settings if provided
	for name, settings := range apiConnectionSettings.OtherConnections {
		if settings.DestinationEndpoint == "" {
			continue
		}

		otherOpts := buildConnectionOptions(settings.Headers, settings.Certificate)

		err = agentModel.SetOtherConnectionSettings(name, settings.DestinationEndpoint, otherOpts...)
		if err != nil {
			return fmt.Errorf("failed to set other connection settings for %s: %w", name, err)
		}
	}

	// Save the updated agent
	err = s.agentUsecase.SaveAgent(ctx, agentModel)
	if err != nil {
		return fmt.Errorf("failed to save agent with connection settings: %w", err)
	}

	// Notify the connected server that the agent has pending messages
	err = s.agentNotificationUsecase.NotifyAgentUpdated(ctx, agentModel)
	if err != nil {
		s.logger.Warn("failed to notify agent update for connection settings",
			slog.String("agentInstanceUID", instanceUID.String()),
			slog.String("error", err.Error()))
		// Don't return error as the connection settings are already saved
	}

	s.logger.Info("connection settings updated for agent",
		"instanceUID", instanceUID.String(),
		"requestedBy", requestedBy)

	return nil
}
