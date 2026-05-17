package client

import (
	"log/slog"
	"strconv"

	"github.com/go-resty/resty/v2"
	"github.com/samber/mo"

	v1auth "github.com/minuk-dev/opampcommander/api/v1/auth"
)

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

// WithBasicAuth exchanges the given credentials for a JWT via /api/v1/auth/basic
// and configures the client to use the resulting Bearer token.
// If the exchange fails (e.g. server not yet ready), the client is left unauthenticated.
func WithBasicAuth(username, password string) OptionFunc {
	return func(cli *Client) {
		var authToken v1auth.AuthnTokenResponse

		res, err := cli.common.Resty.R().
			SetResult(&authToken).
			SetBasicAuth(username, password).
			Get(BasicAuthAPIURL)
		if err != nil || res.IsError() || authToken.Token == "" {
			return
		}

		cli.SetAuthToken(authToken.Token)
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
	// limit caps the number of items returned; nil means no client-side limit.
	limit *int
	// continueToken paginates the request; nil means start from the beginning.
	continueToken *string
	// includeDeleted asks the server to include soft-deleted resources.
	includeDeleted *bool
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

// WithIncludeDeleted sets whether to include deleted resources.
func WithIncludeDeleted(includeDeleted bool) ListOption {
	return ListOptionFunc(func(opt *ListSettings) {
		opt.includeDeleted = &includeDeleted
	})
}

// GetOption is an interface for options that can be applied to get operations.
type GetOption interface {
	Apply(settings *GetSettings)
}

// GetSettings holds the settings for getting a single resource.
type GetSettings struct {
	// includeDeleted asks the server to return the resource even if it is soft-deleted.
	includeDeleted *bool
}

// GetOptionFunc is a function type that implements the GetOption interface.
type GetOptionFunc func(*GetSettings)

// Apply applies the GetOptionFunc to the GetSettings.
func (f GetOptionFunc) Apply(opt *GetSettings) {
	f(opt)
}

// WithGetIncludeDeleted sets whether to include deleted resources for get operations.
func WithGetIncludeDeleted(includeDeleted bool) GetOption {
	return GetOptionFunc(func(opt *GetSettings) {
		opt.includeDeleted = &includeDeleted
	})
}

// applyTo writes the present settings onto the given Resty request as query parameters.
// Absent options leave the request untouched.
func (s ListSettings) applyTo(req *resty.Request) {
	mo.PointerToOption(s.limit).ForEach(func(v int) {
		req.SetQueryParam("limit", strconv.Itoa(v))
	})
	mo.PointerToOption(s.continueToken).ForEach(func(v string) {
		req.SetQueryParam("continue", v)
	})

	if mo.PointerToOption(s.includeDeleted).OrElse(false) {
		req.SetQueryParam("includeDeleted", "true")
	}
}

// applyTo writes the present settings onto the given Resty request as query parameters.
// Absent options leave the request untouched.
func (s GetSettings) applyTo(req *resty.Request) {
	if mo.PointerToOption(s.includeDeleted).OrElse(false) {
		req.SetQueryParam("includeDeleted", "true")
	}
}

// newListSettings folds the given ListOptions into a single ListSettings value.
func newListSettings(opts []ListOption) ListSettings {
	var settings ListSettings
	for _, opt := range opts {
		opt.Apply(&settings)
	}

	return settings
}

// newGetSettings folds the given GetOptions into a single GetSettings value.
func newGetSettings(opts []GetOption) GetSettings {
	var settings GetSettings
	for _, opt := range opts {
		opt.Apply(&settings)
	}

	return settings
}
