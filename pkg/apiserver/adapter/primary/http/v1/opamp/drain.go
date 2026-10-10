package opamp

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"sync"
	"time"
)

// Drain rejects new requests and staggers close frames across the window. The
// caller's deadline bounds both the close handshake and any blocked socket writes.
func (c *Controller) Drain(ctx context.Context, window time.Duration) error {
	c.mu.Lock()
	c.draining = true

	connections := make([]*drainConnection, 0, len(c.connections))
	for conn := range c.connections {
		connections = append(connections, conn)
	}
	c.mu.Unlock()

	var wg sync.WaitGroup
	for _, conn := range connections {
		wg.Go(func() {
			delay := time.Duration(0)
			if window > 0 {
				//nolint:gosec // Jitter needs no cryptographic randomness.
				delay = time.Duration(rand.Int64N(int64(window)) + 1)
			}

			timer := time.NewTimer(delay)
			defer timer.Stop()

			select {
			case <-ctx.Done():
				return
			case <-conn.closed:
				return
			case <-timer.C:
			}

			conn.requestClose()
		})
	}

	done := make(chan struct{})

	go func() {
		wg.Wait()

		for _, conn := range connections {
			<-conn.closed
		}

		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		for _, conn := range connections {
			_ = conn.Close()
		}

		<-done

		return fmt.Errorf("drain OpAMP connections: %w", ctx.Err())
	}
}

type drainResponseWriter struct {
	http.ResponseWriter

	controller *Controller
}

func (w *drainResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	controller := w.controller
	controller.mu.Lock()
	defer controller.mu.Unlock()

	if controller.draining {
		return nil, nil, http.ErrServerClosed
	}

	conn, buffer, err := http.NewResponseController(w.ResponseWriter).Hijack()
	if err != nil {
		return nil, nil, fmt.Errorf("hijack OpAMP connection: %w", err)
	}
	//exhaustruct:ignore
	tracked := &drainConnection{Conn: conn, controller: controller, handshaking: true, closed: make(chan struct{})}
	controller.connections[tracked] = struct{}{}

	return tracked, buffer, nil
}

// opamp-go exposes only an abrupt Disconnect. Guard transport writes so the
// shutdown close frame cannot split a data frame, including gorilla's two-write
// large-payload path. Control frames may appear between message fragments.
type drainConnection struct {
	net.Conn

	controller  *Controller
	mu          sync.Mutex
	remaining   uint64
	handshaking bool
	closing     bool
	closeSent   bool
	once        sync.Once
	closed      chan struct{}
}

func (c *drainConnection) Write(data []byte) (int, error) {
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

func (c *drainConnection) Close() error {
	var err error

	c.once.Do(func() {
		err = c.Conn.Close()
		c.controller.mu.Lock()
		delete(c.controller.connections, c)
		c.controller.mu.Unlock()
		close(c.closed)
	})

	return err //nolint:wrapcheck // Preserve net.Conn's Close contract.
}

func (c *drainConnection) requestClose() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closing = true
	if !c.handshaking && c.remaining == 0 {
		c.writeClose()
	}
}

// writeClose runs under mu. Only the drain deadline force-closes an unresponsive peer.
func (c *drainConnection) writeClose() {
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
