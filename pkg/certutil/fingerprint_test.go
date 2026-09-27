package certutil_test

import (
	"testing"

	"github.com/minuk-dev/opampcommander/pkg/certutil"
)

func TestMatchesSHA256Fingerprint(t *testing.T) {
	t.Parallel()

	fingerprint := certutil.SHA256Fingerprint([]byte("certificate A"))
	if !certutil.MatchesSHA256Fingerprint([]byte("certificate A"), fingerprint) {
		t.Fatal("matching certificate was rejected")
	}

	if certutil.MatchesSHA256Fingerprint([]byte("certificate B"), fingerprint) ||
		certutil.MatchesSHA256Fingerprint([]byte("certificate A"), nil) {
		t.Fatal("nonmatching fingerprint was accepted")
	}
}
