package opampconnection

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type hijackWriter struct {
	*httptest.ResponseRecorder

	conn net.Conn
}

func (w *hijackWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.conn, bufio.NewReadWriter(bufio.NewReader(w.conn), bufio.NewWriter(w.conn)), nil
}

func TestTransport_StopAcceptingWaitsForRegistration(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		transport := NewTransport()

		server, peer := net.Pipe()
		defer func() { _ = peer.Close() }()

		writer := transport.WrapWriter(&hijackWriter{ResponseRecorder: httptest.NewRecorder(), conn: server})
		hijacker, ok := writer.(interface {
			Hijack() (net.Conn, *bufio.ReadWriter, error)
		})
		require.True(t, ok)

		conn, _, err := hijacker.Hijack()
		require.NoError(t, err)

		defer func() { _ = conn.Close() }()

		stopped := make(chan error, 1)
		go func() { stopped <- transport.StopAccepting(t.Context()) }()

		synctest.Wait()
		assert.False(t, transport.Accepting())

		select {
		case <-stopped:
			t.Fatal("accepted upgrade must register before returning")
		default:
		}

		transport.Connected(conn)
		require.NoError(t, <-stopped)
		// Repeated stop calls and close callbacks must not close the admission channel twice.
		require.NoError(t, transport.StopAccepting(t.Context()))
		transport.Connected(conn)
	})
}

func TestTransport_StopAcceptingForceClosesPendingHandshake(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		transport := NewTransport()

		server, peer := net.Pipe()
		defer func() { _ = peer.Close() }()

		writer := &responseWriter{
			ResponseWriter: &hijackWriter{ResponseRecorder: httptest.NewRecorder(), conn: server},
			transport:      transport,
		}
		_, _, err := writer.Hijack()
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()

		require.ErrorIs(t, transport.StopAccepting(ctx), context.DeadlineExceeded)

		_, err = peer.Read(make([]byte, 1))
		require.ErrorIs(t, err, io.EOF)
		_, _, err = writer.Hijack()
		require.Error(t, err, "new upgrades must be rejected during shutdown")
	})
}
