package vo

import (
	"bytes"
	"fmt"
	"hash/fnv"
)

// Hash is a type to manage hash.
type Hash []byte

// NewHash creates a new Hash from the given data.
func NewHash(data []byte) (Hash, error) {
	hash := fnv.New64()

	_, err := hash.Write(data)
	if err != nil {
		return nil, fmt.Errorf("failed to write data to hash: %w", err)
	}

	return Hash(hash.Sum(nil)), nil
}

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
