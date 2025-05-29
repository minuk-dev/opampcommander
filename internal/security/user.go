package security

// User represents a user in the system.
type User struct {
	// Authenticated indicates if the user is authenticated
	Authenticated bool
	// Email is the primary email of the user
	// If the user is not authenticated, this will be nil
	Email *string
}
