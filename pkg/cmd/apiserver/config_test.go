package apiserver_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/minuk-dev/opampcommander/pkg/cmd/apiserver"
)

//nolint:paralleltest // t.Setenv is incompatible with t.Parallel.
func TestConfigView(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(configFile, []byte(`
management:
  log:
    level: warn
auth:
  jwt:
    secret: s3cr3t
`), 0o600)
	require.NoError(t, err)

	t.Setenv("MANAGEMENT_LOG_FORMAT", "json")

	run := func(t *testing.T, extraArgs ...string) (string, map[string]any) {
		t.Helper()

		//exhaustruct:ignore
		cmd := apiserver.NewCommand(apiserver.CommandOption{})

		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetArgs(append([]string{
			"config", "view", "--config", configFile, "--database.type", "mongodb",
		}, extraArgs...))
		require.NoError(t, cmd.Execute())

		var parsed map[string]any
		require.NoError(t, yaml.Unmarshal(out.Bytes(), &parsed))

		return out.String(), parsed
	}

	t.Run("merges flags, config file and env", func(t *testing.T) {
		out, parsed := run(t)

		assert.Contains(t, out, "# config file: "+configFile)
		assert.Contains(t, out, "#   - MANAGEMENT_LOG_FORMAT")

		database, _ := parsed["database"].(map[string]any)
		assert.Equal(t, "mongodb", database["type"])
		assert.Equal(t, "10s", database["connectTimeout"])

		management, _ := parsed["management"].(map[string]any)
		log, _ := management["log"].(map[string]any)
		assert.Equal(t, "warn", log["level"])
		assert.Equal(t, "json", log["format"])

		auth, _ := parsed["auth"].(map[string]any)
		jwt, _ := auth["jwt"].(map[string]any)
		assert.Equal(t, "<redacted>", jwt["secret"])

		oauth2, _ := auth["oauth2"].(map[string]any)
		assert.Empty(t, oauth2["clientSecret"], "empty secrets are not redacted")
	})

	t.Run("show secrets", func(t *testing.T) {
		_, parsed := run(t, "--show-secrets")

		auth, _ := parsed["auth"].(map[string]any)
		jwt, _ := auth["jwt"].(map[string]any)
		assert.Equal(t, "s3cr3t", jwt["secret"])
	})
}
