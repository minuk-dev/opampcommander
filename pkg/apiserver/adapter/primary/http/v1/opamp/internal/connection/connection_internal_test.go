package connection

import (
	"bytes"
	"context"
	"encoding/binary"
	"net"
	"strconv"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/testutil/nettest"
)

func TestConn_Write(t *testing.T) {
	t.Parallel()

	for _, size := range []int{10, 4096, 100000} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			t.Parallel()

			raw := nettest.NewRecordingConnection()
			conn := &connection{Conn: raw, connections: NewConnections(), closed: make(chan struct{})}
			payload := bytes.Repeat([]byte("a"), size)

			header := []byte{0x82, byte(size)}
			if size > 65535 {
				header = make([]byte, 10)
				header[0], header[1] = 0x82, 127
				binary.BigEndian.PutUint64(header[2:], uint64(size))
			} else if size > 125 {
				header = make([]byte, 4)
				header[0], header[1] = 0x82, 126
				binary.BigEndian.PutUint16(header[2:], uint16(size))
			}

			_, err := conn.Write(header)
			require.NoError(t, err)
			conn.requestClose()
			assert.Equal(t, header, raw.Buffer.Bytes(), "close must not interrupt a split frame")

			_, err = conn.Write(payload)
			require.NoError(t, err)

			want := bytes.Join([][]byte{header, payload, []byte("\x88\x02\x03\xe9")}, nil)
			assert.Equal(t, want, raw.Buffer.Bytes())

			_, err = conn.Write([]byte{0x80, 0})
			require.ErrorIs(t, err, net.ErrClosed)
		})
	}
}

func TestConn_CloseGracefullyUnblocksBlockedWrite(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		server, peer := net.Pipe()
		defer func() { _ = peer.Close() }()

		conn := &connection{Conn: server, connections: NewConnections(), closed: make(chan struct{})}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		// The peer never reads, so writing the close frame blocks until force-close.
		err := conn.closeGracefully(ctx)
		require.ErrorIs(t, err, context.DeadlineExceeded)

		select {
		case <-conn.closed:
		default:
			t.Fatal("connection must be force-closed")
		}
	})
}

func TestConn_CloseGracefullyAfterUpgrade(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		raw := nettest.NewRecordingConnection()
		conn := &connection{
			Conn: raw, connections: NewConnections(), handshaking: true, closed: make(chan struct{}),
		}
		handshake := []byte("HTTP/1.1 101 Switching Protocols\r\n\r\n")
		_, err := conn.Write(handshake)
		require.NoError(t, err)

		done := make(chan error, 1)
		go func() { done <- conn.closeGracefully(t.Context()) }()

		synctest.Wait()
		assert.Equal(t, append(handshake, []byte("\x88\x02\x03\xe9")...), raw.Buffer.Bytes())

		select {
		case <-raw.Closed:
			t.Fatal("wait for the peer before closing")
		default:
		}

		require.NoError(t, conn.Close())
		require.NoError(t, <-done)
		require.NoError(t, conn.Close())
	})
}
