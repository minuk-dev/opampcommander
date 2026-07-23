package agentservice

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/utils/clock"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

var _ agentport.HostUsecase = (*HostService)(nil)

// HostService provides operations for managing discovered hosts.
type HostService struct {
	persistence agentport.HostPersistencePort
	clock       clock.PassiveClock
}

// NewHostService creates a new HostService.
func NewHostService(
	persistence agentport.HostPersistencePort,
	passiveClock clock.PassiveClock,
) *HostService {
	return &HostService{
		persistence: persistence,
		clock:       passiveClock,
	}
}

// GetHost implements [agentport.HostUsecase].
func (s *HostService) GetHost(ctx context.Context, id string) (*agentmodel.Host, error) {
	host, err := s.persistence.GetHost(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get host: %w", err)
	}

	return host, nil
}

// ListHosts implements [agentport.HostUsecase].
func (s *HostService) ListHosts(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Host], error) {
	resp, err := s.persistence.ListHosts(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list hosts: %w", err)
	}

	return resp, nil
}

// ObserveAgent implements [agentport.HostUsecase].
//
// Discovery is a read-modify-write, so a concurrent writer (another HA node
// discovering the same host) can make PutHost return [model.ErrConflict]. Rather
// than dropping this agent's association until its next report, the whole cycle is
// retried on conflict so the losing write re-reads the winner's version and merges
// onto it.
func (s *HostService) ObserveAgent(ctx context.Context, agent *agentmodel.Agent) error {
	id := agentmodel.HostIDOf(agent.Metadata.Description)
	if id == "" {
		// The agent reported no host attributes; nothing to discover.
		return nil
	}

	now := s.clock.Now()

	for attempt := 0; ; attempt++ {
		host, err := s.persistence.GetHost(ctx, id)
		if err != nil {
			if !errors.Is(err, model.ErrResourceNotExist) {
				return fmt.Errorf("failed to get host for discovery: %w", err)
			}

			host = agentmodel.NewHost(id, now)
		}

		host.ObserveAgent(agent.Metadata.InstanceUID, agent.Metadata.Description, now)

		_, err = s.persistence.PutHost(ctx, host)
		if err == nil {
			return nil
		}

		if errors.Is(err, model.ErrConflict) && attempt < discoveryObserveConflictRetries {
			continue
		}

		return fmt.Errorf("failed to save discovered host: %w", err)
	}
}

// discoveryObserveConflictRetries bounds how many times a host/container discovery
// read-modify-write is retried when a concurrent writer wins the optimistic-
// concurrency race. A small bound is enough: contention is between the few agents
// on one machine, and any residual loss self-heals on the next agent report.
const discoveryObserveConflictRetries = 3
