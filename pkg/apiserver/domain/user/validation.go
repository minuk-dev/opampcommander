package usermodel

import (
	"errors"
	"fmt"
	"regexp"
)

// MaxUsernameLength bounds a basic-auth username.
const MaxUsernameLength = 128

// ErrInvalidUsername is returned when a username is empty, too long, or contains characters
// outside the allowed set.
var ErrInvalidUsername = errors.New("invalid username")

// usernamePattern is the allowlist for basic-auth usernames: ASCII letters, digits, and a small
// set of separators commonly found in usernames and email-style logins. It deliberately excludes
// whitespace, quotes, and structural characters ('{', '}', '$', etc.), so a username can never
// carry anything that could be mistaken for a query operator or control sequence.
var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9._%+@-]+$`)

// ValidateUsername reports whether username is acceptable for a basic-auth user.
// It enforces a strict allowlist and a length bound; everything else is rejected.
func ValidateUsername(username string) error {
	if username == "" {
		return fmt.Errorf("%w: must not be empty", ErrInvalidUsername)
	}

	if len(username) > MaxUsernameLength {
		return fmt.Errorf("%w: must be at most %d characters", ErrInvalidUsername, MaxUsernameLength)
	}

	if !usernamePattern.MatchString(username) {
		return fmt.Errorf("%w: only letters, digits and the characters . _ %% + @ - are allowed",
			ErrInvalidUsername)
	}

	return nil
}
