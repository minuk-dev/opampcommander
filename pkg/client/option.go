package client

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

// WithBarearToken sets the Bearer token for the client.
func WithBarearToken(barearToken string) OptionFunc {
	return func(c *Client) {
		c.SetAuthToken(barearToken)
	}
}
