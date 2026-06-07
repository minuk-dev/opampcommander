package security

// User represents a user in the system.
type User struct {
	// Authenticated indicates if the user is authenticated
	Authenticated bool
	// Email is the primary email of the user
	// If the user is not authenticated, this will be nil
	Email *string
}

func (user *User) String() string {
	if user == nil {
		return "anonymous"
	}

	if user.Authenticated {
		if user.Email != nil {
			return *user.Email
		}

		return "authenticated;no-email"
	}

	return "unauthenticated"
}
