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
func WithBearerToken(bearerToken string) OptionFunc {
	return func(c *Client) {
		c.SetAuthToken(bearerToken)
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

// ListOption is an interface for options that can be applied to list operations.
type ListOption interface {
	Apply(settings *ListSettings)
}

// ListSettings holds the settings for listing resources.
type ListSettings struct {
	// how many items to return
	// specially, if this is set to 0, it will return all items
	limit *int
	// continue token for pagination
	continueToken *string
}

// ListOptionFunc is a function type that implements the ListOption interface.
type ListOptionFunc func(*ListSettings)

// Apply applies the ListOptionFunc to the ListSettings.
func (f ListOptionFunc) Apply(opt *ListSettings) {
	f(opt)
}

// WithLimit sets the limit for the number of items to return.
func WithLimit(limit int) ListOption {
	if limit <= 0 {
		limit = 100 // default limit
	}

	return ListOptionFunc(func(opt *ListSettings) {
		opt.limit = &limit
	})
}

// WithContinueToken sets the continue token for pagination.
func WithContinueToken(token string) ListOption {
	return ListOptionFunc(func(opt *ListSettings) {
		opt.continueToken = &token
	})
}
