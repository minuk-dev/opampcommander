package apiserver_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/cmd/apiserver"
)

//nolint:paralleltest // t.Setenv is incompatible with t.Parallel.
func TestConfigView(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(configFile, []byte(`
management:
  tls:
    certFile: management.crt
    keyFile: management.key
  log:
    level: warn
tls:
  certFile: server.crt
  keyFile: server.key
  minVersion: "1.2"
  cipherSuites: [TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256]
auth:
  jwt:
    secret: s3cr3t
opamp:
  maxMessageSize: 1024
  enableCompression: true
shutdown:
  drainWindow: 2s
  timeout: 15s
event:
  kafka:
    sendTimeout: 3s
    retryBackoff: 25ms
    retryAttempts: 3
`), 0o600)
	require.NoError(t, err)

	t.Setenv("MANAGEMENT_LOG_FORMAT", "json")
	t.Setenv("SHUTDOWN_TIMEOUT", "20s")
	t.Setenv("EVENT_KAFKA_RETRYBACKOFF", "40ms")
	t.Setenv("EVENT_KAFKA_RETRYATTEMPTS", "5")
	t.Setenv("TLS_MINVERSION", "1.3")
	t.Setenv("MANAGEMENT_TLS_CERTFILE", "env-management.crt")

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

		event, _ := parsed["event"].(map[string]any)
		kafka, _ := event["kafka"].(map[string]any)
		assert.Equal(t, "3s", kafka["sendTimeout"])
		assert.Equal(t, "40ms", kafka["retryBackoff"])
		assert.Equal(t, 5, kafka["retryAttempts"])
	})

	t.Run("OpAMP YAML environment and flags", func(t *testing.T) {
		_, parsed := run(t)
		settings, _ := parsed["opamp"].(map[string]any)
		assert.Equal(t, 1024, settings["maxMessageSize"])
		assert.Equal(t, true, settings["enableCompression"])
		t.Setenv("OPAMP_MAXMESSAGESIZE", "2048")
		_, parsed = run(t)
		settings, _ = parsed["opamp"].(map[string]any)
		assert.Equal(t, 2048, settings["maxMessageSize"])
		_, parsed = run(t, "--opamp.maxMessageSize=-1", "--opamp.enableCompression=false")
		settings, _ = parsed["opamp"].(map[string]any)
		assert.Equal(t, -1, settings["maxMessageSize"])
		assert.Equal(t, false, settings["enableCompression"])
	})

	t.Run("shutdown flags override YAML and environment", func(t *testing.T) {
		_, parsed := run(t)
		shutdown, _ := parsed["shutdown"].(map[string]any)
		assert.Equal(t, "2s", shutdown["drainWindow"])
		assert.Equal(t, "20s", shutdown["timeout"])
		_, parsed = run(t, "--shutdown.timeout", "25s", "--shutdown.drainWindow", "3s")
		shutdown, _ = parsed["shutdown"].(map[string]any)
		assert.Equal(t, "3s", shutdown["drainWindow"])
		assert.Equal(t, "25s", shutdown["timeout"])
	})

	t.Run("TLS flags override YAML and environment", func(t *testing.T) {
		_, parsed := run(t)
		tlsSettings, _ := parsed["tls"].(map[string]any)
		assert.Equal(t, "server.crt", tlsSettings["certFile"])
		assert.Equal(t, "server.key", tlsSettings["keyFile"])
		assert.Equal(t, "1.3", tlsSettings["minVersion"])
		assert.Equal(t, []any{"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256"}, tlsSettings["cipherSuites"])

		management, _ := parsed["management"].(map[string]any)
		managementTLS, _ := management["tls"].(map[string]any)
		assert.Equal(t, "env-management.crt", managementTLS["certFile"])

		_, parsed = run(t, "--tls.minVersion", "1.2", "--management.tls.certFile", "flag-management.crt")
		tlsSettings, _ = parsed["tls"].(map[string]any)
		assert.Equal(t, "1.2", tlsSettings["minVersion"])

		management, _ = parsed["management"].(map[string]any)
		managementTLS, _ = management["tls"].(map[string]any)
		assert.Equal(t, "flag-management.crt", managementTLS["certFile"])
	})

	t.Run("Kafka flags override YAML and environment", func(t *testing.T) {
		_, parsed := run(t, "--event.kafka.retryAttempts", "1", "--event.kafka.sendTimeout", "4s")
		event, _ := parsed["event"].(map[string]any)
		kafka, _ := event["kafka"].(map[string]any)
		assert.Equal(t, 1, kafka["retryAttempts"])
		assert.Equal(t, "4s", kafka["sendTimeout"])
	})

	t.Run("zero Kafka settings use defaults", func(t *testing.T) {
		_, parsed := run(t, "--event.kafka.sendTimeout", "0s",
			"--event.kafka.retryBackoff", "0s", "--event.kafka.retryAttempts", "0")
		event, _ := parsed["event"].(map[string]any)
		kafka, _ := event["kafka"].(map[string]any)
		defaults := config.DefaultKafkaSettings()
		assert.Equal(t, defaults.SendTimeout.String(), kafka["sendTimeout"])
		assert.Equal(t, defaults.RetryBackoff.String(), kafka["retryBackoff"])
		assert.Equal(t, defaults.RetryAttempts, kafka["retryAttempts"])
	})

	t.Run("show secrets", func(t *testing.T) {
		_, parsed := run(t, "--show-secrets")

		auth, _ := parsed["auth"].(map[string]any)
		jwt, _ := auth["jwt"].(map[string]any)
		assert.Equal(t, "s3cr3t", jwt["secret"])
	})
}

func TestCommand_RejectsInvalidKafkaSettingsBeforeStartup(t *testing.T) {
	t.Parallel()

	cmd := apiserver.NewCommand(apiserver.CommandOption{})
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"--config", filepath.Join(t.TempDir(), "missing.yaml"),
		"--event.type", "kafka", "--event.kafka.retryAttempts", "-1"})
	require.ErrorIs(t, cmd.Execute(), config.ErrKafkaSendSettingsInvalid)
}
