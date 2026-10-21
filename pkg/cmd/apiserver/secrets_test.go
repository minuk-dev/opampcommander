package apiserver_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/minuk-dev/opampcommander/pkg/cmd/apiserver"
)

func TestSecretFiles(t *testing.T) {
	for _, key := range []string{
		"auth.admin.password", "auth.basic.pepper", "auth.jwt.secret",
		"auth.oauth2.clientSecret", "auth.oauth2.state.jwt.secret",
		"event.direct.authToken", "liveness.redis.password", "database.endpoints",
	} {
		t.Run(key, func(t *testing.T) {
			env := strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
			t.Setenv(env, "")
			filename := filepath.Join(t.TempDir(), "secret")

			value := " file-secret "
			if key == "database.endpoints" {
				value = "mongodb://user:password@mongo-0:27017,mongo-1:27017/?replicaSet=rs0"
			}

			require.NoError(t, os.WriteFile(filename, []byte(value+"\r\n"), 0o600))
			t.Setenv(env+"_FILE", filename)

			out, parsed, err := secretConfigView(t, "--show-secrets")
			require.NoError(t, err)
			require.Contains(t, out, "#   - "+env+"_FILE")
			require.Equal(t, secretExpected(key, value), secretConfigValue(t, parsed, key))
			out, parsed, err = secretConfigView(t)
			require.NoError(t, err)
			require.NotContains(t, out, value)
			require.Equal(t, "<redacted>", secretConfigValue(t, parsed, key))

			// A higher-priority source must not even try to read the file.
			t.Setenv(env+"_FILE", filename+"-missing")
			t.Setenv(env, "env-secret")
			_, parsed, err = secretConfigView(t, "--show-secrets")
			require.NoError(t, err)
			require.Equal(t, secretExpected(key, "env-secret"), secretConfigValue(t, parsed, key))
			_, parsed, err = secretConfigView(t, "--show-secrets", "--"+key, "flag-secret")
			require.NoError(t, err)
			require.Equal(t, secretExpected(key, "flag-secret"), secretConfigValue(t, parsed, key))
		})
	}
}

func TestSecretFileFailures(t *testing.T) {
	t.Setenv("AUTH_JWT_SECRET", "")

	for _, name := range []string{"missing", "directory", "empty", "newline"} {
		t.Run(name, func(t *testing.T) {
			filename := filepath.Join(t.TempDir(), "secret")

			switch name {
			case "directory":
				require.NoError(t, os.Mkdir(filename, 0o700))
			case "empty":
				require.NoError(t, os.WriteFile(filename, nil, 0o600))
			case "newline":
				require.NoError(t, os.WriteFile(filename, []byte("\n"), 0o600))
			}

			t.Setenv("AUTH_JWT_SECRET_FILE", filename)
			out, _, err := secretConfigView(t)
			require.ErrorContains(t, err, "AUTH_JWT_SECRET_FILE")
			require.Empty(t, out)

			cmd := apiserver.NewCommand(apiserver.CommandOption{})
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			cmd.SetArgs([]string{"--config", filepath.Join(t.TempDir(), "missing.yaml")})
			require.ErrorContains(t, cmd.Execute(), "AUTH_JWT_SECRET_FILE", "startup must fail before preparing the server")
		})
	}
}

func TestSecretFileOverridesYAML(t *testing.T) {
	t.Setenv("AUTH_JWT_SECRET", "")
	filename := filepath.Join(t.TempDir(), "secret")
	require.NoError(t, os.WriteFile(filename, []byte("file-secret"), 0o600))
	configFile := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(configFile, []byte("auth:\n  jwt:\n    secret: yaml-secret\n"), 0o600))
	t.Setenv("AUTH_JWT_SECRET_FILE", "")
	_, parsed, err := secretConfigView(t, "--config", configFile, "--show-secrets")
	require.NoError(t, err)
	require.Equal(t, "yaml-secret", secretConfigValue(t, parsed, "auth.jwt.secret"))
	t.Setenv("AUTH_JWT_SECRET_FILE", filename)
	_, parsed, err = secretConfigView(t, "--config", configFile, "--show-secrets")
	require.NoError(t, err)
	require.Equal(t, "file-secret", secretConfigValue(t, parsed, "auth.jwt.secret"))
	_, parsed, err = secretConfigView(t, "--config", configFile, "--show-secrets", "--auth.jwt.secret=")
	require.NoError(t, err)
	require.Empty(t, secretConfigValue(t, parsed, "auth.jwt.secret"))
}

func secretConfigView(t *testing.T, args ...string) (string, map[string]any, error) {
	t.Helper()

	cmd := apiserver.NewCommand(apiserver.CommandOption{})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs(append([]string{"config", "view", "--config", filepath.Join(t.TempDir(), "missing.yaml")}, args...))
	err := cmd.Execute()

	var parsed map[string]any
	if err == nil {
		require.NoError(t, yaml.Unmarshal(out.Bytes(), &parsed))
	}

	return out.String(), parsed, err
}

func secretConfigValue(t *testing.T, config map[string]any, key string) any {
	t.Helper()

	var value any = config
	for part := range strings.SplitSeq(key, ".") {
		mapping, ok := value.(map[string]any)
		require.True(t, ok)

		value = mapping[part]
	}

	return value
}

func secretExpected(key, value string) any {
	if key == "database.endpoints" {
		return []any{value}
	}

	return value
}
