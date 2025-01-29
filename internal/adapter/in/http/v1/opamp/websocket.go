package opamp

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/open-telemetry/opamp-go/protobufs"
	"google.golang.org/protobuf/proto"
)

type opampWSConnection struct {
	conn         *websocket.Conn
	connMutex    *sync.Mutex
	opampHandler *opampHandler
}

func newWSConnection(conn *websocket.Conn) *opampWSConnection {
	return &opampWSConnection{
		conn:         conn,
		connMutex:    &sync.Mutex{},
		opampHandler: newOpampHandler(),
	}
}

func (w *opampWSConnection) Run(ctx context.Context) error {
	for {
		mt, msgBytes, err := w.conn.ReadMessage()
		if err != nil {
			if !websocket.IsUnexpectedCloseError(err) {
				return fmt.Errorf("cannot read a message from websocket: %w", err)
			}
			// Normal close
			return nil
		}

		if mt != websocket.BinaryMessage {
			return fmt.Errorf("received unexpected message type: %v", mt)
		}

		var request protobufs.AgentToServer

		err = decodeMessage(msgBytes, &request)
		if err != nil {
			// todo:
			// log error
			continue
		}

		// Handle the message.
		err = w.opampHandler.handleAgentToServer(ctx, &request)
		if err != nil {
			// todo: log error
			continue
		}

		// Fetch the response.
		response, err := w.opampHandler.fetchServerToAgent(ctx)
		if err != nil {
			// todo: log error
			continue
		}

		// Send the response.
		err = w.Send(ctx, response)
		if err != nil {
			// todo: log error
			continue
		}
	}
}

func (w *opampWSConnection) Close() error {
	w.connMutex.Lock()
	defer w.connMutex.Unlock()

	return w.conn.Close()
}

func (w *opampWSConnection) Send(_ context.Context, message *protobufs.ServerToAgent) error {
	w.connMutex.Lock()
	defer w.connMutex.Unlock()

	bytes, err := proto.Marshal(message)
	if err != nil {
		return err
	}

	err = w.conn.WriteMessage(websocket.BinaryMessage, bytes)
	if err != nil {
		return err
	}

	return nil
}

// Message header is currently uint64 zero value.
const wsMsgHeader = uint64(0)

func decodeMessage(bytes []byte, msg proto.Message) error {
	// Message header is optional until the end of grace period that ends Feb 1, 2023.
	// Check if the header is present.
	if len(bytes) > 0 && bytes[0] == 0 {
		// New message format. The Protobuf message is preceded by a zero byte header.
		// Decode the header.
		header, n := binary.Uvarint(bytes)
		if header != wsMsgHeader {
			return errors.New("unexpected non-zero header")
		}
		// Skip the header. It really is just a single zero byte for now.
		bytes = bytes[n:]
	}
	// If no header was present (the "if" check above), then this is the old
	// message format. No header is present.

	// Decode WebSocket message as a Protobuf message.
	err := proto.Unmarshal(bytes, msg)
	if err != nil {
		return err
	}

	return nil
}
