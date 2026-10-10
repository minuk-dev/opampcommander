package agentservice

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
)

var _ agentport.ConnectionShutdownUsecase = (*ConnectionShutdownService)(nil)

// ConnectionShutdownService owns the policy for closing this server's connections.
type ConnectionShutdownService struct {
	store     agentport.ConnectionStore
	transport agentport.ConnectionTransportPort
}

// NewConnectionShutdownService creates the server-local connection shutdown use case.
func NewConnectionShutdownService(
	store agentport.ConnectionStore, transport agentport.ConnectionTransportPort,
) *ConnectionShutdownService {
	return &ConnectionShutdownService{store: store, transport: transport}
}

// CloseLocalConnections stops admission, snapshots this server's Store, and
// staggers persistent connection closes. HTTP requests drain with the HTTP server.
func (s *ConnectionShutdownService) CloseLocalConnections(ctx context.Context, window time.Duration) error {
	admissionErr := s.transport.StopAccepting(ctx)

	connections, err := s.store.ListConnections(ctx)
	if err != nil {
		return errors.Join(admissionErr, fmt.Errorf("list local connections for shutdown: %w", err))
	}

	results := make(chan error, len(connections))

	var wg sync.WaitGroup

	for _, conn := range connections {
		if conn.Type != agentmodel.ConnectionTypeWebSocket {
			continue
		}

		wg.Go(func() {
			delay := time.Duration(0)
			if window > 0 {
				//nolint:gosec // Reconnect jitter needs no cryptographic randomness.
				delay = time.Duration(rand.Int64N(int64(window)) + 1)
			}

			timer := time.NewTimer(delay)
			defer timer.Stop()

			select {
			case <-timer.C:
			case <-ctx.Done():
			}
			// Even with an expired context the adapter must force-close this connection.
			err := s.transport.CloseConnection(ctx, conn.ID)
			if err != nil {
				results <- fmt.Errorf("close local connection %s: %w", conn.UID, err)
			}
		})
	}

	wg.Wait()
	close(results)

	shutdownErr := admissionErr
	for err := range results {
		shutdownErr = errors.Join(shutdownErr, err)
	}

	return shutdownErr
}
