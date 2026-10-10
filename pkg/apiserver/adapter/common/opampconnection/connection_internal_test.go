package opampconnection

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net"
	"strconv"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/open-telemetry/opamp-go/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingConnection struct {
	net.Conn

	buffer bytes.Buffer
	once   sync.Once
	closed chan struct{}
}

func (c *recordingConnection) Write(data []byte) (int, error) {
	count, _ := c.buffer.Write(data)

	return count, nil
}
func (c *recordingConnection) Close() error {
	c.once.Do(func() { close(c.closed) })

	return nil
}

func TestDrainConnection_Write(t *testing.T) {
	t.Parallel()

	for _, size := range []int{10, 4096, 100000} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			t.Parallel()

			transport := NewTransport()
			raw := &recordingConnection{closed: make(chan struct{})}
			conn := &connection{Conn: raw, transport: transport, closed: make(chan struct{})}
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
			assert.Equal(t, header, raw.buffer.Bytes(), "close must not interrupt a split frame")

			_, err = conn.Write(payload)
			require.NoError(t, err)

			want := bytes.Join([][]byte{header, payload, []byte("\x88\x02\x03\xe9")}, nil)
			assert.Equal(t, want, raw.buffer.Bytes())

			_, err = conn.Write([]byte{0x80, 0})
			require.ErrorIs(t, err, net.ErrClosed)
		})
	}
}

func TestFrameSize_RejectsShortHeaders(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"empty", nil},
		{"one byte", []byte{0x82}},
		{"short 16-bit length", []byte{0x82, 126}},
		{"short 64-bit length", []byte{0x82, 127}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := frameSize(tc.data)
			require.ErrorIs(t, err, io.ErrShortBuffer)
		})
	}
}

func TestTransport_CloseConnectionUnblocksBlockedWrite(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		transport := NewTransport()

		server, peer := net.Pipe()
		defer func() { _ = peer.Close() }()

		conn := &connection{Conn: server, transport: transport, closed: make(chan struct{})}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		// The peer never reads, so writing the close frame blocks until force-close.
		err := transport.CloseConnection(ctx, &testConnection{netConn: conn})
		require.ErrorIs(t, err, context.DeadlineExceeded)

		select {
		case <-conn.closed:
		default:
			t.Fatal("connection must be force-closed")
		}
	})
}

type connectionMethods = types.Connection

type testConnection struct {
	connectionMethods

	netConn net.Conn
}

func (c *testConnection) Connection() net.Conn { return c.netConn }
