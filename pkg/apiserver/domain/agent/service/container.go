//nolint:dupl // Host and Container discovery services intentionally share this shape.
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

var _ agentport.ContainerUsecase = (*ContainerService)(nil)

// ContainerService provides operations for managing discovered containers.
type ContainerService struct {
	persistence agentport.ContainerPersistencePort
	clock       clock.PassiveClock
}

// NewContainerService creates a new ContainerService.
func NewContainerService(
	persistence agentport.ContainerPersistencePort,
	passiveClock clock.PassiveClock,
) *ContainerService {
	return &ContainerService{
		persistence: persistence,
		clock:       passiveClock,
	}
}

// GetContainer implements [agentport.ContainerUsecase].
func (s *ContainerService) GetContainer(ctx context.Context, id string) (*agentmodel.Container, error) {
	container, err := s.persistence.GetContainer(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get container: %w", err)
	}

	return container, nil
}

// ListContainers implements [agentport.ContainerUsecase].
func (s *ContainerService) ListContainers(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Container], error) {
	resp, err := s.persistence.ListContainers(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	return resp, nil
}

// ObserveAgent implements [agentport.ContainerUsecase].
func (s *ContainerService) ObserveAgent(ctx context.Context, agent *agentmodel.Agent) error {
	id := agentmodel.ContainerIDOf(agent.Metadata.Description)
	if id == "" {
		// The agent reported no container attributes; nothing to discover.
		return nil
	}

	now := s.clock.Now()

	container, err := s.persistence.GetContainer(ctx, id)
	if err != nil {
		if !errors.Is(err, model.ErrResourceNotExist) {
			return fmt.Errorf("failed to get container for discovery: %w", err)
		}

		container = agentmodel.NewContainer(id, now)
	}

	container.ObserveAgent(agent.Metadata.InstanceUID, agent.Metadata.Description, now)

	_, err = s.persistence.PutContainer(ctx, container)
	if err != nil {
		return fmt.Errorf("failed to save discovered container: %w", err)
	}

	return nil
}
