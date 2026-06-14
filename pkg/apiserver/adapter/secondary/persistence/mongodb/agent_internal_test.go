package mongodb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrefixUpperBound(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		prefix    string
		wantBound string
		wantOK    bool
	}{
		{name: "simple ascii increments last byte", prefix: "abc", wantBound: "abd", wantOK: true},
		{name: "uuid prefix", prefix: "12345678-1234", wantBound: "12345678-1235", wantOK: true},
		{name: "trailing hex f rolls the byte, not the alphabet", prefix: "abcdef", wantBound: "abcdeg", wantOK: true},
		{name: "trailing 0xFF carries to previous byte", prefix: "ab\xff", wantBound: "ac", wantOK: true},
		{name: "empty has no finite bound", prefix: "", wantBound: "", wantOK: false},
		{name: "all 0xFF has no finite bound", prefix: "\xff\xff", wantBound: "", wantOK: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			bound, ok := prefixUpperBound(tc.prefix)
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.wantBound, bound)
		})
	}
}

// TestPrefixUpperBound_RangeContainsPrefixMatches asserts the core invariant the
// SearchAgents range scan relies on: for a non-empty prefix, every string that
// starts with the prefix sorts in [prefix, upperBound) under byte-wise ordering,
// and a string that does not start with the prefix falls outside it.
func TestPrefixUpperBound_RangeContainsPrefixMatches(t *testing.T) {
	t.Parallel()

	const prefix = "abcd"

	bound, ok := prefixUpperBound(prefix)
	assert.True(t, ok)

	// Matches: prefix itself and any extension are within [prefix, bound).
	for _, match := range []string{"abcd", "abcd1234-5678", "abcdffff"} {
		assert.GreaterOrEqual(t, match, prefix)
		assert.Less(t, match, bound)
	}

	// Non-matches sort outside the range.
	assert.Less(t, "abcc", prefix)          // below the lower bound
	assert.GreaterOrEqual(t, "abce", bound) // at/above the upper bound
}
