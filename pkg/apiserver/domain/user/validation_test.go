package usermodel_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	usermodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user"
)

func TestValidateUsername(t *testing.T) {
	t.Parallel()

	valid := []string{"guest", "admin", "alice.bob", "user_1", "a+b", "user@example.com", "a-b%c"}
	for _, u := range valid {
		require.NoError(t, usermodel.ValidateUsername(u), "want %q valid", u)
	}

	invalid := []string{
		"",                       // empty
		"a b",                    // space
		"name\twith\ttabs",       // whitespace
		`{"$ne": null}`,          // mongo operator document text
		"bob$gt",                 // structural char
		"quote'or'1",             // quote
		strings.Repeat("a", 129), // too long
	}
	for _, u := range invalid {
		require.ErrorIs(t, usermodel.ValidateUsername(u), usermodel.ErrInvalidUsername, "want %q invalid", u)
	}
}
