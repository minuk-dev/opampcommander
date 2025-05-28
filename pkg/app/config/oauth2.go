package config

import "time"

// OAuthSettings holds the configuration settings for GitHub OAuth2 authentication.
type OAuthSettings struct {
	// ClientID is the OAuth2 client ID for GitHub authentication.
	ClientID string
	// Secret is the OAuth2 client secret for GitHub authentication.
	Secret string
	// CallbackURL is the URL to which GitHub will redirect after authentication.
	CallbackURL string
	// JWTSettings holds the JWT configuration settings.
	// This is used for the state parameter in OAuth2 authentication.
	JWTSettings JWTSettings
}

// JWTSettings holds the configuration settings for JSON Web Tokens (JWT).
type JWTSettings struct {
	SigningKey string
	Issuer     string
	Expiration time.Duration
	Audience   []string
}
