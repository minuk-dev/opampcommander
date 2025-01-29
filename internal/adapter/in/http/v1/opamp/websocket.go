package opamp

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/open-telemetry/opamp-go/protobufs"
	"google.golang.org/protobuf/proto"
)

var ErrUnexpectedNonZeroHeader = errors.New("unexpected non-zero header")

type UnexpectedMessageTypeError struct {
	MessageType int
}

func (e *UnexpectedMessageTypeError) Error() string {
	return fmt.Sprintf("unexpected message type: %v", e.MessageType)
}

type opampWSConnection struct {
	conn         *websocket.Conn
	connMutex    *sync.Mutex
	opampHandler *opampHandler
	logger       *slog.Logger
}

func newWSConnection(conn *websocket.Conn, logger *slog.Logger) *opampWSConnection {
	return &opampWSConnection{
		conn:         conn,
		connMutex:    &sync.Mutex{},
		opampHandler: newOpampHandler(),
		logger:       logger,
	}
}

func (w *opampWSConnection) Run(ctx context.Context) error {
	for {
		messageType, msgBytes, err := w.conn.ReadMessage()
		if err != nil {
			if !websocket.IsUnexpectedCloseError(err) {
				return fmt.Errorf("cannot read a message from websocket: %w", err)
			}
			// Normal close
			return nil
		}

		if messageType != websocket.BinaryMessage {
			return &UnexpectedMessageTypeError{MessageType: messageType}
		}

		var request protobufs.AgentToServer

		err = decodeMessage(msgBytes, &request)
		if err != nil {
			w.logger.Warn("cannot decode message", "error", err.Error())

			continue
		}

		// Handle the message.
		err = w.opampHandler.handleAgentToServer(ctx, &request)
		if err != nil {
			w.logger.Warn("cannot handle message", "error", err.Error())

			continue
		}

		// Fetch the response.
		response, err := w.opampHandler.fetchServerToAgent(ctx)
		if err != nil {
			w.logger.Warn("cannot fetch response", "error", err.Error())

			continue
		}

		// Send the response.
		err = w.Send(ctx, response)
		if err != nil {
			w.logger.Warn("cannot send response", "error", err.Error())

			continue
		}
	}
}

func (w *opampWSConnection) Close() error {
	w.connMutex.Lock()
	defer w.connMutex.Unlock()

	err := w.conn.Close()
	if err != nil {
		return fmt.Errorf("cannot close websocket connection: %w", err)
	}

	return nil
}

func (w *opampWSConnection) Send(_ context.Context, message *protobufs.ServerToAgent) error {
	w.connMutex.Lock()
	defer w.connMutex.Unlock()

	bytes, err := proto.Marshal(message)
	if err != nil {
		return fmt.Errorf("cannot marshal message: %w", err)
	}

	err = w.conn.WriteMessage(websocket.BinaryMessage, bytes)
	if err != nil {
		return fmt.Errorf("cannot write message to websocket: %w", err)
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
			return ErrUnexpectedNonZeroHeader
		}
		// Skip the header. It really is just a single zero byte for now.
		bytes = bytes[n:]
	}
	// If no header was present (the "if" check above), then this is the old
	// message format. No header is present.

	// Decode WebSocket message as a Protobuf message.
	err := proto.Unmarshal(bytes, msg)
	if err != nil {
		return fmt.Errorf("cannot unmarshal message: %w", err)
	}

	return nil
}
