package config

import "time"

// AuthSettings holds the authentication settings for the application.
type AuthSettings struct {
	// JWTSettings holds the configuration settings for JSON Web Tokens (JWT).
	JWTSettings JWTSettings `json:"jwtSettings" mapstructure:"jwtSettings" yaml:"jwtSettings"`
	// AdminSettings holds the configuration settings for admin authentication.
	AdminSettings AdminSettings `json:"adminSettings" mapstructure:"adminSettings" yaml:"adminSettings"`
	// OAuthSettings holds the configuration settings for OAuth2 authentication.
	OAuthSettings *OAuthSettings
}

// AdminSettings holds the configuration settings for admin authentication.
// This is used for basic authentication of the admin user.
type AdminSettings struct {
	// Username is the username for the admin user.
	Username string
	// Password is the password for the admin user.
	Password string
	// Email is the email address for the admin user.
	// This is used in jwt claims.
	Email string
}

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
