// Package opampconnection adapts OpAMP WebSocket transport operations.
package opampconnection

import (
	"encoding/binary"
	"io"
	"net"
	"sync"
)

// opamp-go exposes only an abrupt Disconnect. Guard transport writes so the
// shutdown close frame cannot split a data frame, including gorilla's two-write
// large-payload path. Control frames may appear between message fragments.
type connection struct {
	net.Conn

	transport   *Transport
	mu          sync.Mutex
	remaining   uint64
	handshaking bool
	closing     bool
	closeSent   bool
	once        sync.Once
	closed      chan struct{}
}

func (c *connection) Write(data []byte) (int, error) {
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

func (c *connection) Close() error {
	var err error

	c.once.Do(func() {
		err = c.Conn.Close()
		c.transport.connected(c)
		close(c.closed)
	})

	return err //nolint:wrapcheck // Preserve net.Conn's Close contract.
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
	// FIN + unmasked close, two-byte payload: 1001 (Going Away).
	_, err := io.WriteString(c.Conn, "\x88\x02\x03\xe9")
	if err != nil {
		_ = c.Close()
	}
}

//nolint:mnd // RFC 6455 frame header lengths and length markers.
func frameSize(data []byte) (uint64, error) {
	// shortcut: gorilla writes each complete server frame header in its first Write;
	// remove this transport guard when opamp-go exposes a control-frame API.
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
