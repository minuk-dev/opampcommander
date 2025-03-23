package opamp

func WithCompression(enable bool) Option {
	return func(c *Controller) {
		c.enableCompression = enable
	}
}
