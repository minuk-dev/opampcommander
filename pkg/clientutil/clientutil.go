// Package clientutil provides some util functions to set client
package clientutil

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/spf13/afero"

	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/filecache"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/config"
	"github.com/minuk-dev/opampcommander/pkg/opampctl/configutil"
)

const (
	// TokenPrefix is the prefix used for cached tokens in the file system.
	TokenPrefix = "token"

	// BarearTokenKey is the key used to store the barear token in the file cache.
	BarearTokenKey = "barear"
)

// NewClient creates a new authenticated Client.
func NewClient(
	config *config.GlobalConfig,
) (*client.Client, error) {
	cli, err := NewAuthedClient(config)
	if err != nil {
		if errors.Is(err, filecache.ErrNoCachedKey) {
			cli, err = NewAuthedClientByIssuingTokenInCli(config)
			if err != nil {
				return nil, fmt.Errorf("failed to create authenticated client: %w", err)
			}

			return cli, nil
		}

		return nil, fmt.Errorf("failed to create authenticated client: %w", err)
	}

	return cli, nil
}

// NewAuthedClient creates a new authenticated OpAMP client using the cached barear token.
// It retrieves the token from the file cache and initializes the client with it.
func NewAuthedClient(
	config *config.GlobalConfig,
) (*client.Client, error) {
	endpoint := configutil.GetCurrentOpAMPCommanderEndpoint(config)
	cacheDir := configutil.GetCurrentCacheDir(config)
	user := configutil.GetCurrentUser(config)

	filesystem := afero.NewOsFs()
	tokenPrefix := TokenPath(user.Name)
	filecache := filecache.New(cacheDir, tokenPrefix, filesystem)

	barearToken, err := filecache.Get(BarearTokenKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get barear token from cache: %w", err)
	}

	cli := client.New(
		endpoint,
		client.WithBarearToken(string(barearToken)),
		client.WithLogger(config.Log.Logger),
		client.WithVerbose(config.Log.Level == slog.LevelDebug),
	)

	_, err = cli.AuthService.GetInfo() // no need to use info. It's just to check if the client is authenticated
	if err != nil {
		var httpErr *client.ResponseError
		if errors.As(err, &httpErr) {
			if httpErr.StatusCode == http.StatusUnauthorized {
				return nil, fmt.Errorf("unauthorized: please re-authenticate: %w", err)
			}
		}

		return nil, fmt.Errorf("failed to get auth info: %w", err)
	}

	return cli, nil
}

// NewAuthedClientByIssuingTokenInCli creates a new authenticated OpAMP client by issuing a token in the CLI.
// It supports different authentication methods such as basic auth, manual token input, and GitHub device.
func NewAuthedClientByIssuingTokenInCli(
	conf *config.GlobalConfig,
) (*client.Client, error) {
	endpoint := configutil.GetCurrentOpAMPCommanderEndpoint(conf)
	cacheDir := configutil.GetCurrentCacheDir(conf)
	user := configutil.GetCurrentUser(conf)

	filesystem := afero.NewOsFs()
	tokenPrefix := TokenPath(user.Name)
	filecache := filecache.New(cacheDir, tokenPrefix, filesystem)

	cli := client.New(endpoint,
		client.WithLogger(conf.Log.Logger),
		client.WithVerbose(conf.Log.Level == slog.LevelDebug),
	)

	barearToken, err := getAuthToken(cli, user, conf.Output)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth token: %w", err)
	}

	err = cacheToken(filecache, BarearTokenKey, barearToken)
	if err != nil {
		// Log the error but do not return it, as we still want to return the client.
		conf.Log.Logger.Warn("failed to cache barear token", "error", err)
	}

	cli.SetAuthToken(string(barearToken))

	return cli, nil
}

// getAuthToken handles the authentication process based on the user's auth type.
func getAuthToken(cli *client.Client, user *config.User, writer io.Writer) ([]byte, error) {
	switch user.Auth.Type {
	case config.AuthTypeBasic:
		return getAuthTokenByBasicAuth(cli, user.Auth.Username, user.Auth.Password)
	case config.AuthTypeManual:
		return []byte(user.Auth.BearerToken), nil
	case config.AuthTypeGithub:
		return getAuthTokenByGithub(cli, writer)
	default:
		return nil, &UnsupportedAuthMethodError{Method: user.Auth.Type}
	}
}

// getAuthTokenByBasicAuth retrieves the auth token using basic authentication.
func getAuthTokenByBasicAuth(cli *client.Client, username, password string) ([]byte, error) {
	authToken, err := cli.AuthService.GetAuthTokenByBasicAuth(username, password)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth token by basic auth: %w", err)
	}

	return []byte(authToken.Token), nil
}

// getAuthTokenByGithub handles the GitHub device authentication flow.
func getAuthTokenByGithub(cli *client.Client, writer io.Writer) ([]byte, error) {
	deviceAuthResponse, err := cli.AuthService.GetDeviceAuthToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get device auth token: %w", err)
	}

	_, err = fmt.Fprintf(writer,
		"Please open the following URL in your browser: %s\n"+
			"Then enter the user code: %s\n"+
			"Wait for authentication...\n",
		deviceAuthResponse.VerificationURI,
		deviceAuthResponse.UserCode,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to write device auth instructions: %w", err)
	}

	authToken, err := cli.AuthService.ExchangeDeviceAuthToken(
		deviceAuthResponse.DeviceCode,
		deviceAuthResponse.Expiry,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange device auth token: %w", err)
	}

	return []byte(authToken.Token), nil
}

// cacheToken caches the authentication token for future use.
func cacheToken(filecache *filecache.FileCache, key string, token []byte) error {
	err := filecache.Set(key, token)
	if err != nil {
		return fmt.Errorf("failed to cache token: %w", err)
	}

	return nil
}

// TokenPath constructs the file path for the cached token of a specific user.
func TokenPath(username string) string {
	return filepath.Join(username, TokenPrefix)
}
