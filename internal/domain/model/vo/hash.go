package vo

import "bytes"

// Hash is a type to manage hash.
type Hash []byte

func (h Hash) String() string {
	return string(h)
}

func (h Hash) Bytes() []byte {
	return h
}

func (h Hash) Equal(other Hash) bool {
	return bytes.Equal(h, other)
}
