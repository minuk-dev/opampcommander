// Package nettest provides test doubles for network connections.
package nettest

import (
	"bytes"
	"net"
	"sync"
)

// RecordingConnection records writes and signals its first close.
// Only Write and Close are implemented; other methods use the embedded Conn.
type RecordingConnection struct {
	net.Conn

	Buffer bytes.Buffer
	once   sync.Once
	Closed chan struct{}
}

// NewRecordingConnection creates a connection with an initialized close signal.
func NewRecordingConnection() *RecordingConnection {
	//exhaustruct:ignore
	return &RecordingConnection{Closed: make(chan struct{})}
}

// Write appends data to Buffer.
func (c *RecordingConnection) Write(data []byte) (int, error) {
	count, _ := c.Buffer.Write(data)

	return count, nil
}

// Close signals Closed once.
func (c *RecordingConnection) Close() error {
	c.once.Do(func() { close(c.Closed) })

	return nil
}
