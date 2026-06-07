package security

import "errors"

var (
	// ErrInvalidState is returned when the state parameter is invalid.
	ErrInvalidState = errors.New("invalid state parameter")
	// ErrStateExpired is returned when the state parameter has expired.
	ErrStateExpired = errors.New("state parameter has expired")
	// ErrInvalidToken is returned when the provided token is invalid.
	ErrInvalidToken = errors.New("invalid token")
	// ErrInvalidEmail is returned when the email in the token claims is invalid.
	ErrInvalidEmail = errors.New("invalid email in token claims")
	// ErrInvalidTokenClaims is returned when the token claims are invalid.
	ErrInvalidTokenClaims = errors.New("invalid token claims")
	// ErrTokenExpired is returned when the token has expired.
	ErrTokenExpired = errors.New("token has expired")
	// ErrInvalidUsernameOrPassword is returned when the provided username or password is invalid.
	ErrInvalidUsernameOrPassword = errors.New("invalid username or password")
	// ErrNoPrimaryEmailFound is returned when no primary email is found in the user's emails.
	ErrNoPrimaryEmailFound = errors.New("no primary verified email found")
	// ErrOAuth2ClientCreationFailed is returned when the OAuth2 client creation fails.
	ErrOAuth2ClientCreationFailed = errors.New("failed to create OAuth2 client")
)

// UnsupportedTokenTypeError is returned when the token type is not supported.
type UnsupportedTokenTypeError struct {
	TokenType string
}

func (e *UnsupportedTokenTypeError) Error() string {
	return "unsupported token type: " + e.TokenType
}
