package view_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/cmd/opampctl/config/view"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
)

func TestViewRedactsSecretsWithoutChangingConfig(t *testing.T) {
	t.Parallel()
	cfg := config.NewDefaultGlobalConfig(t.TempDir())
	cfg.Users[0].Auth.Password = "private-password"
	cfg.Users = append(cfg.Users, config.User{
		Name: "manual", Auth: config.Auth{ManualAuth: config.ManualAuth{BearerToken: "private-token"}},
	})

	for _, show := range []bool{false, true} {
		name := "redacted"
		if show {
			name = "show-secrets"
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd := view.NewCommand(view.CommandOptions{GlobalConfig: cfg})

			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetArgs([]string{})

			if show {
				cmd.SetArgs([]string{"--show-secrets"})
			}

			require.NoError(t, cmd.Execute())

			for _, secret := range []string{"private-password", "private-token"} {
				if show {
					require.Contains(t, out.String(), secret)
				} else {
					require.NotContains(t, out.String(), secret)
					require.Contains(t, out.String(), "<redacted>")
				}
			}

			require.Equal(t, "private-password", cfg.Users[0].Auth.Password)
			require.Equal(t, "private-token", cfg.Users[1].Auth.BearerToken)
		})
	}
}
