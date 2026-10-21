package apiserver

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var errEmptySecretFile = errors.New("secret file is empty")

// resolveSecretFiles preserves flag > env > file > YAML/default precedence.
func (opt *CommandOption) resolveSecretFiles(cmd *cobra.Command) error {
	var databaseURI string

	opt.secretFileEnvs = nil
	for _, secret := range []struct {
		key    string
		target *string
	}{
		{"auth.admin.password", &opt.Auth.Admin.Password},
		{"auth.basic.pepper", &opt.Auth.Basic.Pepper},
		{"auth.jwt.secret", &opt.Auth.JWT.Secret},
		{"auth.oauth2.clientSecret", &opt.Auth.OAuth2.ClientSecret},
		{"auth.oauth2.state.jwt.secret", &opt.Auth.OAuth2.State.JWT.Secret},
		{"event.direct.authToken", &opt.Event.Direct.AuthToken},
		{"liveness.redis.password", &opt.Liveness.Redis.Password},
		{"database.endpoints", &databaseURI},
	} {
		env := strings.ToUpper(strings.ReplaceAll(secret.key, ".", "_"))

		filename := os.Getenv(env + "_FILE")
		if filename == "" || cmd.Flags().Changed(secret.key) || os.Getenv(env) != "" {
			continue
		}

		content, err := os.ReadFile(filename) //nolint:gosec // Path explicitly supplied by the server operator.
		if err != nil {
			return fmt.Errorf("read %s_FILE: %w", env, err)
		}

		// Strip the line ending added by secret provisioning tools, preserving spaces.
		value := strings.TrimSuffix(strings.TrimSuffix(string(content), "\n"), "\r")
		if value == "" {
			return fmt.Errorf("%s_FILE: %w", env, errEmptySecretFile)
		}

		*secret.target = value

		opt.secretFileEnvs = append(opt.secretFileEnvs, env+"_FILE")
	}

	if databaseURI != "" {
		// One complete URI, including commas in replica-set host lists.
		opt.Database.Endpoints = []string{databaseURI}
	}

	return nil
}
