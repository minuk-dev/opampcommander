package opamp

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/open-telemetry/opamp-go/protobufs"
	"google.golang.org/protobuf/proto"

	"github.com/minuk-dev/minuk-apiserver/internal/domain/model"
)

var ErrUnexpectedNonZeroHeader = errors.New("unexpected non-zero header")

type UnexpectedMessageTypeError struct {
	MessageType int
}

func (e *UnexpectedMessageTypeError) Error() string {
	return fmt.Sprintf("unexpected message type: %v", e.MessageType)
}

type connectionAdapter struct {
	// domain
	conn *model.Connection

	// internal utils
	logger *slog.Logger
}

func newConnectionAdapter(logger *slog.Logger) *connectionAdapter {
	adapter := &connectionAdapter{
		conn:   nil,
		logger: logger,
	}

	return adapter
}

func (w *connectionAdapter) Run(ctx context.Context, wsConn *websocket.Conn) error {
	agentToServerChan, err := w.init(ctx, wsConn)
	if err != nil {
		return fmt.Errorf("cannot initialize connection: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case agentToServer := <-agentToServerChan:
			err := w.conn.HandleAgentToServer(ctx, agentToServer)
			if err != nil {
				return fmt.Errorf("cannot handle agentToServer message: %w", err)
			}
		case serverToAgent := <-w.conn.Watch():
			err := send(ctx, wsConn, serverToAgent)
			if err != nil {
				return fmt.Errorf("cannot send a message: %w", err)
			}
		}
	}
	// unreachable
}

func receive(_ context.Context, wsConn *websocket.Conn) (*protobufs.AgentToServer, error) {
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

func send(_ context.Context, wsConn *websocket.Conn, message *protobufs.ServerToAgent) error {
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

func (w *connectionAdapter) init(ctx context.Context, wsConn *websocket.Conn) (<-chan *protobufs.AgentToServer, error) {
	request, err := receive(ctx, wsConn)
	if err != nil {
		return nil, fmt.Errorf("cannot receive a message: %w", err)
	}

	instanceUID, err := uuid.ParseBytes(request.GetInstanceUid())
	if err != nil {
		return nil, fmt.Errorf("cannot parse instance UUID: %w", err)
	}

	w.conn = model.NewConnection(instanceUID)
	err = w.conn.HandleAgentToServer(ctx, request)
	if err != nil {
		w.logger.Warn("cannot handle agentToServer message", "error", err.Error())
	}

	agentToServerChan := make(chan *protobufs.AgentToServer)

	go func(ctx context.Context) {
		defer close(agentToServerChan)

		for {
			agentToServer, err := receive(ctx, wsConn)
			if err != nil {
				w.logger.Warn("cannot receive a message", "error", err.Error())

				return
			}
			agentToServerChan <- agentToServer
		}
	}(ctx)

	return agentToServerChan, nil
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
