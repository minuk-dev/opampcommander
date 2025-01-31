package wsprotobufutil

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/open-telemetry/opamp-go/protobufs"
	"google.golang.org/protobuf/proto"
)

var ErrUnexpectedNonZeroHeader = errors.New("unexpected non-zero header")

type UnexpectedMessageTypeError struct {
	MessageType int
}

// Message header is currently uint64 zero value.
const wsMsgHeader = uint64(0)

func (e *UnexpectedMessageTypeError) Error() string {
	return fmt.Sprintf("unexpected message type: %v", e.MessageType)
}

func Receive(_ context.Context, wsConn *websocket.Conn) (*protobufs.AgentToServer, error) {
	messageType, msgBytes, err := wsConn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("cannot read message from websocket: %w", err)
	}

	if messageType != websocket.BinaryMessage {
		return nil, &UnexpectedMessageTypeError{MessageType: messageType}
	}

	var request protobufs.AgentToServer

	err = decodeMessage(msgBytes, &request)
	if err != nil {
		return nil, fmt.Errorf("cannot decode message: %w", err)
	}

	return &request, nil
}

func Send(_ context.Context, wsConn *websocket.Conn, message *protobufs.ServerToAgent) error {
	bytes, err := proto.Marshal(message)
	if err != nil {
		return fmt.Errorf("cannot marshal message: %w", err)
	}

	err = wsConn.WriteMessage(websocket.BinaryMessage, bytes)
	if err != nil {
		return fmt.Errorf("cannot write message to websocket: %w", err)
	}

	return nil
}

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
