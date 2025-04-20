package vo

import "bytes"

// Hash is a type to manage hash.
type Hash []byte

// String returns the string representation of the hash.
func (h Hash) String() string {
	return string(h)
}

// Bytes returns the byte slice of the hash.
func (h Hash) Bytes() []byte {
	return h
}

// Equal compares two Hash values for equality.
func (h Hash) Equal(other Hash) bool {
	return bytes.Equal(h, other)
}
