package client

import "log/slog"

// Option provides a way to configure the opampcommander API client.
type Option interface {
	Apply(client *Client)
}

// OptionFunc is a function that applies an option to the Client.
type OptionFunc func(*Client)

// Apply applies the option to the Client.
func (f OptionFunc) Apply(c *Client) {
	f(c)
}

// WithBearerToken sets the Bearer token for the client.
func WithBearerToken(barearToken string) OptionFunc {
	return func(c *Client) {
		c.SetAuthToken(barearToken)
	}
}

// WithVerbose enables verbose logging for the client.
func WithVerbose(verbose bool) OptionFunc {
	return func(c *Client) {
		c.SetVerbose(verbose)
	}
}

// WithLogger sets the logger for the client.
func WithLogger(logger *slog.Logger) OptionFunc {
	return func(c *Client) {
		c.SetLogger(logger)
	}
}
