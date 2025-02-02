package hashutil

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
)

func Hash(data any) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	err := enc.Encode(data)
	if err != nil {
		return nil, fmt.Errorf("hashutil: failed to encode data: %w", err)
	}

	hash := sha256.Sum256(buf.Bytes())

	return hash[:], nil
}
