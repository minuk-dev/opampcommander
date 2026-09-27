// Package certutil provides helpers for X.509 certificate bytes.
package certutil

import (
	"bytes"
	"crypto/sha256"
)

// SHA256Fingerprint returns the fingerprint of a DER-encoded certificate.
func SHA256Fingerprint(certDER []byte) []byte {
	sum := sha256.Sum256(certDER)

	return sum[:]
}

// MatchesSHA256Fingerprint reports whether certDER has the given fingerprint.
func MatchesSHA256Fingerprint(certDER, fingerprint []byte) bool {
	return bytes.Equal(SHA256Fingerprint(certDER), fingerprint)
}
