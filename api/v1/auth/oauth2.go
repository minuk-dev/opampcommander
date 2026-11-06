package auth

const (
	// OAuth2ErrorParam carries the OAuth 2.0 error code.
	OAuth2ErrorParam = "error"
	// OAuth2ErrorDescriptionParam carries the public explanation of an OAuth 2.0 error.
	OAuth2ErrorDescriptionParam = "error_description"
)

const (
	// OAuth2ErrorAccessDenied indicates that authorization was denied.
	OAuth2ErrorAccessDenied = "access_denied"
	// OAuth2ErrorInvalidRequest indicates a missing or invalid request parameter.
	OAuth2ErrorInvalidRequest = "invalid_request"
	// OAuth2ErrorServerError indicates that the server could not complete authorization.
	OAuth2ErrorServerError = "server_error"
)
