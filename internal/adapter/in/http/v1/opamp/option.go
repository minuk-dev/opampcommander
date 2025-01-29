package opamp

import (
	"github.com/gorilla/websocket"

	"github.com/minuk-dev/minuk-apiserver/internal/domain/port"
)

func WithCompression(enable bool) Option {
	return func(c *Controller) {
		c.wsUpgrader = websocket.Upgrader{
			HandshakeTimeout:  0,
			ReadBufferSize:    0,
			WriteBufferSize:   0,
			WriteBufferPool:   nil,
			Subprotocols:      nil,
			Error:             nil,
			CheckOrigin:       nil,
			EnableCompression: enable,
		}
	}
}

func WithConnectionUsecase(connectionUsecase port.ConnectionUsecase) Option {
	return func(c *Controller) {
		c.connectionUsecase = connectionUsecase
	}
}
