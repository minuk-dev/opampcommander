package connection

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/testutil/nettest"
)

type hijackWriter struct {
	*httptest.ResponseRecorder

	conn net.Conn
}

func (w *hijackWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.conn, bufio.NewReadWriter(bufio.NewReader(w.conn), bufio.NewWriter(w.conn)), nil
}

func TestConnections_ShutdownIncludesPendingUpgrade(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		connections := NewConnections()
		raw := nettest.NewRecordingConnection()
		writer := &responseWriter{
			ResponseWriter: &hijackWriter{ResponseRecorder: httptest.NewRecorder(), conn: raw},
			connections:    connections,
		}
		conn, _, err := writer.Hijack()
		require.NoError(t, err)

		done := make(chan error, 1)
		go func() { done <- connections.Shutdown(t.Context(), time.Second) }()

		synctest.Wait()
		assert.False(t, connections.Accepting())
		assert.Empty(t, raw.Buffer.Bytes(), "jitter must delay the close request")
		time.Sleep(time.Second)
		assert.Empty(t, raw.Buffer.Bytes(), "close frame must follow the HTTP upgrade response")

		handshake := []byte("HTTP/1.1 101 Switching Protocols\r\n\r\n")
		_, err = conn.Write(handshake)
		require.NoError(t, err)
		assert.Equal(t, append(handshake, []byte("\x88\x02\x03\xe9")...), raw.Buffer.Bytes())
		require.NoError(t, conn.Close())
		require.NoError(t, <-done)
		assert.Empty(t, connections.active)

		_, _, err = writer.Hijack()
		require.ErrorIs(t, err, http.ErrServerClosed)
		require.NoError(t, connections.Shutdown(t.Context(), 0))
	})
}

func TestConnections_ShutdownForceClosesStalledUpgrade(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		connections := NewConnections()

		server, peer := net.Pipe()
		defer func() { _ = peer.Close() }()

		writer := &responseWriter{
			ResponseWriter: &hijackWriter{ResponseRecorder: httptest.NewRecorder(), conn: server},
			connections:    connections,
		}
		_, _, err := writer.Hijack()
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()

		require.ErrorIs(t, connections.Shutdown(ctx, 0), context.DeadlineExceeded)

		_, err = peer.Read(make([]byte, 1))
		require.ErrorIs(t, err, io.EOF)
		assert.Empty(t, connections.active)
	})
}

func TestConnections_ShutdownClosesAllAfterDeadline(t *testing.T) {
	t.Parallel()

	connections := NewConnections()
	sockets := make([]*nettest.RecordingConnection, 0, 2)

	for range 2 {
		raw := nettest.NewRecordingConnection()
		writer := &responseWriter{
			ResponseWriter: &hijackWriter{ResponseRecorder: httptest.NewRecorder(), conn: raw},
			connections:    connections,
		}
		_, _, err := writer.Hijack()
		require.NoError(t, err)

		sockets = append(sockets, raw)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, connections.Shutdown(ctx, time.Hour), context.Canceled)

	for _, raw := range sockets {
		select {
		case <-raw.Closed:
		default:
			t.Fatal("every socket must be closed even after the deadline")
		}
	}

	assert.Empty(t, connections.active)
}
