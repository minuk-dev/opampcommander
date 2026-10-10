package agentservice

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
)

// CloseLocalConnections stops admission, snapshots this server's Store, and
// staggers persistent connection closes. HTTP requests drain with the HTTP server.
func (s *Service) CloseLocalConnections(ctx context.Context, window time.Duration) error {
	admissionErr := s.transport.StopAccepting(ctx)

	connections, err := s.connectionStore.ListConnections(ctx)
	if err != nil {
		return errors.Join(admissionErr, fmt.Errorf("list local connections for shutdown: %w", err))
	}

	var (
		group    sync.WaitGroup
		closeMu  sync.Mutex
		closeErr error
	)

	for _, conn := range connections {
		if conn.Type != agentmodel.ConnectionTypeWebSocket {
			continue
		}

		group.Go(func() {
			delay := time.Duration(0)
			if window > 0 {
				//nolint:gosec // Reconnect jitter needs no cryptographic randomness.
				delay = time.Duration(rand.Int64N(int64(window)) + 1)
			}

			select {
			case <-time.After(delay):
			case <-ctx.Done():
			}
			// Even with an expired context the adapter must force-close this connection.
			err := s.transport.CloseConnection(ctx, conn.ID)
			if err != nil {
				closeMu.Lock()
				closeErr = errors.Join(closeErr, fmt.Errorf("close local connection %s: %w", conn.UID, err))
				closeMu.Unlock()
			}
		})
	}

	group.Wait()

	return errors.Join(admissionErr, closeErr)
}
