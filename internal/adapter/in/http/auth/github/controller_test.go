package github_test

import (
	"testing"

	github "github.com/minuk-dev/opampcommander/internal/adapter/in/http/auth/github"
)

func TestValidateRedirect(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		rawURL  string
		allowed []string
		wantErr bool
	}{
		// Loopback hosts are always accepted, regardless of allowlist.
		{"loopback IPv4", "http://127.0.0.1:3000/cb", nil, false},
		{"loopback IPv6", "http://[::1]:3000/cb", nil, false},
		{"localhost", "http://localhost:3000/cb", nil, false},
		{"https loopback", "https://127.0.0.1/cb", nil, false},

		// Scheme must be http/https.
		{"bad scheme", "javascript:alert(1)", nil, true},
		{"file scheme", "file:///etc/passwd", nil, true},

		// Non-loopback host without allowlist → rejected.
		{"non-loopback without allowlist", "https://evil.com/cb", nil, true},
		{
			"non-loopback with mismatching allowlist",
			"https://evil.com/cb",
			[]string{"opampcommander-alpha.minuk.dev"},
			true,
		},

		// Non-loopback host that is on the allowlist → accepted.
		{
			"non-loopback in allowlist",
			"https://opampcommander-alpha.minuk.dev/login/github/callback",
			[]string{"opampcommander-alpha.minuk.dev"},
			false,
		},
		{
			"non-loopback in allowlist with port",
			"https://opampcommander-alpha.minuk.dev:8443/cb",
			[]string{"opampcommander-alpha.minuk.dev"},
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := github.ValidateRedirect(tc.rawURL, tc.allowed)
			if (err != nil) != tc.wantErr {
				t.Fatalf(
					"ValidateRedirect(%q, %v) = %v; wantErr=%v",
					tc.rawURL, tc.allowed, err, tc.wantErr,
				)
			}
		})
	}
}
