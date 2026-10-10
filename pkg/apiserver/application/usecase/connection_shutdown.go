package usecase

import (
	"context"
	"time"
)

// ConnectionShutdownUsecase closes the connections owned by this server.
type ConnectionShutdownUsecase interface {
	CloseLocalConnections(ctx context.Context, window time.Duration) error
}
