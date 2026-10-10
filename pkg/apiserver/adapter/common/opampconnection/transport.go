package opampconnection

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/open-telemetry/opamp-go/server/types"

	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
)

var _ agentport.ConnectionTransportPort = (*Transport)(nil)

var errUnmanagedConnection = errors.New("connection has no managed OpAMP WebSocket transport")

// Transport owns admission and WebSocket close mechanics, never shutdown policy.
// Only in-progress upgrades are tracked here; ConnectionStore owns live sessions.
type Transport struct {
	mu       sync.Mutex
	stopping bool
	pending  map[*connection]struct{}
	admitted chan struct{}
}

// NewTransport creates the transport shared by the controller and shutdown use case.
func NewTransport() *Transport {
	return &Transport{
		mu: sync.Mutex{}, stopping: false,
		pending: make(map[*connection]struct{}), admitted: make(chan struct{}),
	}
}

// Accepting reports whether the endpoint may accept new OpAMP requests.
func (t *Transport) Accepting() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	return !t.stopping
}

// WrapWriter registers upgrades before handing their transport to opamp-go.
func (t *Transport) WrapWriter(writer http.ResponseWriter) http.ResponseWriter {
	return &responseWriter{ResponseWriter: writer, transport: t}
}

// Connected completes admission after the application has registered the session.
func (t *Transport) Connected(conn net.Conn) {
	managed, ok := conn.(*connection)
	if ok {
		t.connected(managed)
	}
}

// StopAccepting waits for admitted upgrades to reach the application's Store.
func (t *Transport) StopAccepting(ctx context.Context) error {
	t.mu.Lock()
	if !t.stopping {
		t.stopping = true
		if len(t.pending) == 0 {
			close(t.admitted)
		}
	}
	t.mu.Unlock()

	select {
	case <-t.admitted:
		return nil
	case <-ctx.Done():
		t.mu.Lock()

		pending := make([]*connection, 0, len(t.pending))
		for conn := range t.pending {
			pending = append(pending, conn)
		}
		t.mu.Unlock()

		for _, conn := range pending {
			_ = conn.Close()
		}

		return fmt.Errorf("stop OpAMP admission: %w", ctx.Err())
	}
}

// CloseConnection sends Going Away and bounds blocked writes and the peer's reply.
func (t *Transport) CloseConnection(ctx context.Context, id any) error {
	conn, ok := id.(types.Connection)
	if !ok {
		return errUnmanagedConnection
	}

	managed, ok := conn.Connection().(*connection)
	if !ok {
		return errUnmanagedConnection
	}

	stop := context.AfterFunc(ctx, func() { _ = managed.Close() })
	defer stop()

	managed.requestClose()
	<-managed.closed

	err := ctx.Err()
	if err != nil {
		return fmt.Errorf("close OpAMP connection: %w", err)
	}

	return nil
}

func (t *Transport) connected(conn *connection) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, ok := t.pending[conn]; !ok {
		return
	}

	delete(t.pending, conn)

	if t.stopping && len(t.pending) == 0 {
		close(t.admitted)
	}
}

type responseWriter struct {
	http.ResponseWriter

	transport *Transport
}

func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	transport := w.transport
	transport.mu.Lock()
	defer transport.mu.Unlock()

	if transport.stopping {
		return nil, nil, http.ErrServerClosed
	}

	conn, buffer, err := http.NewResponseController(w.ResponseWriter).Hijack()
	if err != nil {
		return nil, nil, fmt.Errorf("hijack OpAMP connection: %w", err)
	}
	//exhaustruct:ignore
	managed := &connection{Conn: conn, transport: transport, handshaking: true, closed: make(chan struct{})}
	transport.pending[managed] = struct{}{}

	return managed, buffer, nil
}
