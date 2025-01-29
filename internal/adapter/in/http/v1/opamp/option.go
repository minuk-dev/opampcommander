package opamp

import "github.com/gorilla/websocket"

func WithCompression(enable bool) Option {
	return func(c *Controller) {
		c.wsUpgrader = websocket.Upgrader{
			EnableCompression: enable,
		}
	}
}
