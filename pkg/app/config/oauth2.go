package config

// OAuthSettings holds the configuration settings for GitHub OAuth2 authentication.
type OAuthSettings struct {
	ClientID    string
	Secret      string
	CallbackURL string
	SigningKey  string
}
