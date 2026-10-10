// Package websocketutil provides server-side WebSocket transport helpers.
package websocketutil

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
)

// Conn guards server writes so a close frame cannot split a data frame,
// including gorilla's two-write large-payload path.
type Conn struct {
	net.Conn

	onClose     func()
	mu          sync.Mutex
	remaining   uint64
	handshaking bool
	closing     bool
	closeSent   bool
	once        sync.Once
	closed      chan struct{}
}

// NewServerConn wraps a hijacked connection before its HTTP upgrade response is
// written. onClose, if non-nil, runs once when the underlying connection closes.
func NewServerConn(conn net.Conn, onClose func()) *Conn {
	//exhaustruct:ignore
	return &Conn{Conn: conn, onClose: onClose, handshaking: true, closed: make(chan struct{})}
}

// CloseGracefully sends Going Away, waits for the caller's read loop to Close
// after the peer's reply, and force-closes when ctx expires, including blocked writes.
func (c *Conn) CloseGracefully(ctx context.Context) error {
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

// Write serializes writes and inserts a requested close at a frame boundary.
func (c *Conn) Write(data []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closeSent {
		return 0, net.ErrClosed
	}

	if !c.handshaking && c.remaining == 0 {
		size, err := frameSize(data)
		if err != nil {
			return 0, err
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

// Close immediately closes the underlying connection and invokes onClose once.
func (c *Conn) Close() error {
	var err error

	c.once.Do(func() {
		err = c.Conn.Close()
		if c.onClose != nil {
			c.onClose()
		}
		close(c.closed)
	})

	return err //nolint:wrapcheck // Preserve net.Conn's Close contract.
}

func (c *Conn) requestClose() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closing = true
	if !c.handshaking && c.remaining == 0 {
		c.writeClose()
	}
}

// writeClose runs under mu. Only the drain deadline force-closes an unresponsive peer.
func (c *Conn) writeClose() {
	if c.closeSent {
		return
	}

	c.closeSent = true
	// FIN + unmasked close, two-byte payload: 1001 (Going Away).
	_, err := io.WriteString(c.Conn, "\x88\x02\x03\xe9")
	if err != nil {
		_ = c.Close()
	}
}

//nolint:mnd // RFC 6455 frame header lengths and length markers.
func frameSize(data []byte) (uint64, error) {
	// shortcut: gorilla writes each complete server frame header in its first Write;
	// use the WebSocket library's control-frame API directly when available.
	if len(data) < 2 {
		return 0, io.ErrShortBuffer
	}

	size := uint64(data[1] & 0x7f)
	switch size {
	case 126:
		if len(data) < 4 {
			return 0, io.ErrShortBuffer
		}

		return 4 + uint64(binary.BigEndian.Uint16(data[2:4])), nil
	case 127:
		if len(data) < 10 {
			return 0, io.ErrShortBuffer
		}

		return 10 + binary.BigEndian.Uint64(data[2:10]), nil
	default:
		return 2 + size, nil
	}
}
