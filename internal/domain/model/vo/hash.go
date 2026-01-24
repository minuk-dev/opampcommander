package vo

import (
	"bytes"
	"encoding/json"
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

func NewHashFromAny(data any) (Hash, error) {
	var byteData []byte

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data to json: %w", err)
	}

	byteData = jsonData

	return NewHash(byteData)
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

// IsZero returns true if the hash is zero (empty).
func (h Hash) IsZero() bool {
	return len(h) == 0
}
