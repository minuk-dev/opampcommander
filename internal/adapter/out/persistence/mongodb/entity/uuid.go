package entity

import "github.com/google/uuid"

// parseUUIDOrNil parses a UUID string, returning uuid.Nil on failure.
func parseUUIDOrNil(s string) uuid.UUID {
	uid, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}

	return uid
}
