// Package connectionshutdown exposes the local connection shutdown use case.
package connectionshutdown

import (
	"context"
	"fmt"
	"time"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/usecase"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
)

var _ usecase.ConnectionShutdownUsecase = (*Service)(nil)

// Service delegates local connection shutdown to the domain policy.
type Service struct {
	connections agentport.ConnectionShutdownUsecase
}

// New creates the application service.
func New(connections agentport.ConnectionShutdownUsecase) *Service {
	return &Service{connections: connections}
}

// CloseLocalConnections closes this server's connections within the supplied deadline.
func (s *Service) CloseLocalConnections(ctx context.Context, window time.Duration) error {
	err := s.connections.CloseLocalConnections(ctx, window)
	if err != nil {
		return fmt.Errorf("shutdown local connections: %w", err)
	}

	return nil
}
