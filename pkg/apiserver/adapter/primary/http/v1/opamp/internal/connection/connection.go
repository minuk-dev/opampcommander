package connection

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/minuk-dev/opampcommander/pkg/websocketutil"
)

// connection guards server writes so a close frame cannot split a data frame,
// including gorilla's two-write large-payload path.
type connection struct {
	net.Conn

	connections *Connections
	mu          sync.Mutex
	remaining   uint64
	handshaking bool
	closing     bool
	closeSent   bool
	once        sync.Once
	closed      chan struct{}
}

// Write serializes writes and inserts a requested close at a frame boundary.
func (c *connection) Write(data []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closeSent {
		return 0, net.ErrClosed
	}

	if !c.handshaking && c.remaining == 0 {
		// shortcut: gorilla writes each complete server frame header in its first Write;
		// use the WebSocket library's control-frame API directly when available.
		size, err := websocketutil.FrameSize(data)
		if err != nil {
			return 0, fmt.Errorf("read WebSocket frame size: %w", err)
		}

		c.remaining = size
	}

	count, err := c.Conn.Write(data)
	if c.handshaking {
		c.handshaking = false
	} else {
		c.remaining -= uint64(count)
	}

	if err == nil && c.closing && c.remaining == 0 {
		c.writeClose()
	}

	return count, err //nolint:wrapcheck // Preserve net.Conn's Write contract.
}

// Close closes the underlying connection and removes it from the controller once.
func (c *connection) Close() error {
	var err error

	c.once.Do(func() {
		err = c.Conn.Close()
		c.connections.remove(c)
		close(c.closed)
	})

	return err //nolint:wrapcheck // Preserve net.Conn's Close contract.
}

// closeGracefully sends Going Away, waits for the caller's read loop to Close
// after the peer's reply, and force-closes when ctx expires, including blocked writes.
func (c *connection) closeGracefully(ctx context.Context) error {
	stop := context.AfterFunc(ctx, func() { _ = c.Close() })
	defer stop()

	c.requestClose()
	<-c.closed

	err := ctx.Err()
	if err != nil {
		return fmt.Errorf("close WebSocket connection: %w", err)
	}

	return nil
}

func (c *connection) requestClose() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closing = true
	if !c.handshaking && c.remaining == 0 {
		c.writeClose()
	}
}

// writeClose runs under mu. Only the drain deadline force-closes an unresponsive peer.
func (c *connection) writeClose() {
	if c.closeSent {
		return
	}

	c.closeSent = true

	_, err := c.Conn.Write(websocketutil.GoingAwayFrame())
	if err != nil {
		_ = c.Close()
	}
}
