// Package auth provides the authentication api for the opampcommander application
package auth

// AuthnTokenResponse defines the response structure for authentication token requests.
type AuthnTokenResponse struct {
	// Token is the authentication token.
	Token string `json:"token"`
}

// OAuth2AuthCodeURLResponse defines the response structure for OAuth2 authorization URL requests.
// It contains the URL that the client should redirect to for OAuth2 authentication.
type OAuth2AuthCodeURLResponse struct {
	// URL is the OAuth2 authorization URL.
	URL string `json:"url"`
}
