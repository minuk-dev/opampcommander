package opamp

import (
	"context"
	"fmt"
	"time"
)

// CloseLocalConnections delegates shutdown to the connection business logic.
func (s *Service) CloseLocalConnections(ctx context.Context, window time.Duration) error {
	err := s.connectionUsecase.CloseLocalConnections(ctx, window)
	if err != nil {
		return fmt.Errorf("shutdown local connections: %w", err)
	}

	return nil
}
