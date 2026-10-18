// Package connection manages the OpAMP controller's WebSocket connections.
package connection

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"net/http"
	"sync"
	"time"
)

// Connections owns hijacked sockets, including upgrades still in progress.
type Connections struct {
	mu       sync.Mutex
	stopping bool
	active   map[*connection]struct{}
}

// NewConnections creates an empty set of controller connections.
func NewConnections() *Connections {
	return &Connections{mu: sync.Mutex{}, stopping: false, active: make(map[*connection]struct{})}
}

// Accepting reports whether the controller may accept new requests.
func (c *Connections) Accepting() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return !c.stopping
}

// WrapWriter registers sockets as soon as they are hijacked for a WebSocket upgrade.
func (c *Connections) WrapWriter(writer http.ResponseWriter) http.ResponseWriter {
	return &responseWriter{ResponseWriter: writer, connections: c}
}

// Shutdown stops admission and staggers close handshakes across drainWindow.
// The context deadline force-closes unresponsive peers and incomplete upgrades.
func (c *Connections) Shutdown(ctx context.Context, drainWindow time.Duration) error {
	c.mu.Lock()
	c.stopping = true

	sockets := make([]*connection, 0, len(c.active))
	for conn := range c.active {
		sockets = append(sockets, conn)
	}
	c.mu.Unlock()

	var (
		group    sync.WaitGroup
		closeMu  sync.Mutex
		closeErr error
	)

	for _, conn := range sockets {
		group.Go(func() {
			delay := time.Duration(0)
			if drainWindow > 0 {
				//nolint:gosec // Reconnect jitter needs no cryptographic randomness.
				delay = time.Duration(rand.Int64N(int64(drainWindow)) + 1)
			}

			select {
			case <-time.After(delay):
			case <-ctx.Done():
			}

			err := conn.closeGracefully(ctx)
			if err != nil {
				closeMu.Lock()
				closeErr = errors.Join(closeErr, err)
				closeMu.Unlock()
			}
		})
	}

	group.Wait()

	return closeErr
}

func (c *Connections) remove(conn *connection) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.active, conn)
}

type responseWriter struct {
	http.ResponseWriter

	connections *Connections
}

func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	connections := w.connections
	connections.mu.Lock()
	defer connections.mu.Unlock()

	if connections.stopping {
		return nil, nil, http.ErrServerClosed
	}

	conn, buffer, err := http.NewResponseController(w.ResponseWriter).Hijack()
	if err != nil {
		return nil, nil, fmt.Errorf("hijack OpAMP connection: %w", err)
	}

	//exhaustruct:ignore
	managed := &connection{Conn: conn, connections: connections, handshaking: true, closed: make(chan struct{})}
	connections.active[managed] = struct{}{}

	return managed, buffer, nil
}
