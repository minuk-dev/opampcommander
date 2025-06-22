package clientutil

// static check to ensure UnsupportedAuthMethodError implements the error interface.
var _ error = (*UnsupportedAuthMethodError)(nil)

// UnsupportedAuthMethodError represents an error for unsupported authentication methods.
type UnsupportedAuthMethodError struct {
	Method string
}

// Error implements the error interface for UnsupportedAuthMethodError.
func (e *UnsupportedAuthMethodError) Error() string {
	return "unsupported authentication method: " + e.Method
}
