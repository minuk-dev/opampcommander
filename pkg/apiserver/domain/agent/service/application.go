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

var _ agentport.ApplicationUsecase = (*ApplicationService)(nil)

// ApplicationService provides operations for managing discovered applications.
type ApplicationService struct {
	persistence agentport.ApplicationPersistencePort
	clock       clock.PassiveClock
}

// NewApplicationService creates a new ApplicationService.
func NewApplicationService(
	persistence agentport.ApplicationPersistencePort,
	passiveClock clock.PassiveClock,
) *ApplicationService {
	return &ApplicationService{
		persistence: persistence,
		clock:       passiveClock,
	}
}

// GetApplication implements [agentport.ApplicationUsecase].
func (s *ApplicationService) GetApplication(ctx context.Context, id string) (*agentmodel.Application, error) {
	application, err := s.persistence.GetApplication(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get application: %w", err)
	}

	return application, nil
}

// ListApplications implements [agentport.ApplicationUsecase].
func (s *ApplicationService) ListApplications(
	ctx context.Context,
	options *model.ListOptions,
) (*model.ListResponse[*agentmodel.Application], error) {
	resp, err := s.persistence.ListApplications(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list applications: %w", err)
	}

	return resp, nil
}

// ObserveAgent implements [agentport.ApplicationUsecase].
//
// Like host discovery, this is a read-modify-write retried on
// [model.ErrConflict] so a concurrent writer (another HA node discovering the same
// application) does not drop this agent's association.
func (s *ApplicationService) ObserveAgent(ctx context.Context, agent *agentmodel.Agent) error {
	id := agentmodel.ApplicationIDOf(agent.Metadata.Description)
	if id == "" {
		// The agent reported no application attributes; nothing to discover.
		return nil
	}

	now := s.clock.Now()

	for attempt := 0; ; attempt++ {
		application, err := s.persistence.GetApplication(ctx, id)
		if err != nil {
			if !errors.Is(err, model.ErrResourceNotExist) {
				return fmt.Errorf("failed to get application for discovery: %w", err)
			}

			application = agentmodel.NewApplication(id, now)
		}

		application.ObserveAgent(agent.Metadata.InstanceUID, agent.Metadata.Description, now)

		_, err = s.persistence.PutApplication(ctx, application)
		if err == nil {
			return nil
		}

		if errors.Is(err, model.ErrConflict) && attempt < discoveryObserveConflictRetries {
			continue
		}

		return fmt.Errorf("failed to save discovered application: %w", err)
	}
}
