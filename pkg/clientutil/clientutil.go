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

	// BearerTokenKey is the key used to store the bearer token in the file cache.
	BearerTokenKey = "bearer"
	// RefreshTokenKey is the key used to store the refresh token in the file cache.
	RefreshTokenKey = "refresh"
)

var (
	// ErrUnauthorized is returned when the client is unauthorized.
	ErrUnauthorized = errors.New("unauthorized")
	// ErrNoEndpoint is returned when no endpoint is configured for the current context.
	ErrNoEndpoint = errors.New("no endpoint configured for current context: " +
		"run 'opampctl context use <name>' or set a cluster endpoint in your config")
)

// NewClient creates a new authenticated Client.
// Resolution order: cached access token → cached refresh token → interactive login.
func NewClient(
	config *config.GlobalConfig,
) (*client.Client, error) {
	if configutil.GetCurrentOpAMPCommanderEndpoint(config) == "" {
		return nil, ErrNoEndpoint
	}

	cli, err := NewAuthedClient(config)
	if err == nil {
		return cli, nil
	}

	if !errors.Is(err, filecache.ErrNoCachedKey) && !errors.Is(err, ErrUnauthorized) {
		return nil, fmt.Errorf("failed to create authenticated client: %w", err)
	}

	cli, refreshErr := NewAuthedClientByRefreshToken(config)
	if refreshErr == nil {
		return cli, nil
	}

	config.Log.Logger.Debug("refresh token unavailable, falling back to interactive login",
		slog.String("reason", refreshErr.Error()))

	cli, err = NewAuthedClientByIssuingTokenInCli(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create authenticated client: %w", err)
	}

	return cli, nil
}

// NewUnauthenticatedClient creates a new unauthenticated Client.
// Returns nil if no endpoint is configured for the current context.
func NewUnauthenticatedClient(
	config *config.GlobalConfig,
) *client.Client {
	endpoint := configutil.GetCurrentOpAMPCommanderEndpoint(config)
	if endpoint == "" {
		return nil
	}

	cli := client.New(
		endpoint,
		client.WithLogger(config.Log.Logger),
		client.WithVerbose(config.Log.Level == slog.LevelDebug),
	)

	return cli
}

// NewAuthedClient creates a new authenticated OpAMP client using the cached bearer token.
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

	bearerToken, err := filecache.Get(BearerTokenKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get bearer token from cache: %w", err)
	}

	cli := client.New(
		endpoint,
		client.WithBearerToken(string(bearerToken)),
		client.WithLogger(config.Log.Logger),
		client.WithVerbose(config.Log.Level == slog.LevelDebug),
	)

	_, err = cli.AuthService.GetInfo() // no need to use info. It's just to check if the client is authenticated
	if err != nil {
		var httpErr *client.ResponseError
		if errors.As(err, &httpErr) {
			if httpErr.StatusCode == http.StatusUnauthorized {
				return nil, ErrUnauthorized
			}
		}

		return nil, fmt.Errorf("failed to get auth info: %w", err)
	}

	return cli, nil
}

// NewAuthedClientByRefreshToken attempts to mint a fresh access token from the cached refresh token.
// Returns ErrUnauthorized if the refresh token is missing, expired, or rejected by the server.
func NewAuthedClientByRefreshToken(
	conf *config.GlobalConfig,
) (*client.Client, error) {
	endpoint := configutil.GetCurrentOpAMPCommanderEndpoint(conf)
	cacheDir := configutil.GetCurrentCacheDir(conf)
	user := configutil.GetCurrentUser(conf)

	cache := filecache.New(cacheDir, TokenPath(user.Name), afero.NewOsFs())

	refreshToken, err := cache.Get(RefreshTokenKey)
	if err != nil {
		return nil, fmt.Errorf("no cached refresh token: %w", err)
	}

	cli := client.New(endpoint,
		client.WithLogger(conf.Log.Logger),
		client.WithVerbose(conf.Log.Level == slog.LevelDebug),
	)

	resp, err := cli.AuthService.Refresh(string(refreshToken))
	if err != nil {
		var httpErr *client.ResponseError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusUnauthorized {
			return nil, ErrUnauthorized
		}

		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	persistTokens(cache, conf.Log.Logger, resp.Token, resp.RefreshToken)
	cli.SetAuthToken(resp.Token)

	return cli, nil
}

// NewAuthedClientByIssuingTokenInCli creates a new authenticated OpAMP client by issuing a token in the CLI.
// It supports different authentication methods such as basic auth, manual token input, and GitHub.
func NewAuthedClientByIssuingTokenInCli(
	conf *config.GlobalConfig,
) (*client.Client, error) {
	endpoint := configutil.GetCurrentOpAMPCommanderEndpoint(conf)
	cacheDir := configutil.GetCurrentCacheDir(conf)
	user := configutil.GetCurrentUser(conf)

	cache := filecache.New(cacheDir, TokenPath(user.Name), afero.NewOsFs())

	cli := client.New(endpoint,
		client.WithLogger(conf.Log.Logger),
		client.WithVerbose(conf.Log.Level == slog.LevelDebug),
	)

	tokens, err := getAuthTokens(cli, user, conf.Output, conf.Log.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth token: %w", err)
	}

	persistTokens(cache, conf.Log.Logger, tokens.access, tokens.refresh)
	cli.SetAuthToken(tokens.access)

	return cli, nil
}

// authTokens carries the freshly minted access/refresh token pair from an interactive login.
type authTokens struct {
	access  string
	refresh string
}

// getAuthTokens runs the configured login flow for the user and returns the resulting tokens.
func getAuthTokens(cli *client.Client, user *config.User, writer io.Writer, logger *slog.Logger) (authTokens, error) {
	switch user.Auth.Type {
	case config.AuthTypeBasic:
		return getTokensByBasicAuth(cli, user.Auth.Username, user.Auth.Password)
	case config.AuthTypeManual:
		return authTokens{access: user.Auth.BearerToken, refresh: ""}, nil
	case config.AuthTypeGithub:
		return getTokensByGithub(cli, user.Auth.Flow, writer, logger)
	default:
		return authTokens{}, &UnsupportedAuthMethodError{Method: user.Auth.Type}
	}
}

func getTokensByBasicAuth(cli *client.Client, username, password string) (authTokens, error) {
	resp, err := cli.AuthService.GetAuthTokenByBasicAuth(username, password)
	if err != nil {
		return authTokens{}, fmt.Errorf("failed to get auth token by basic auth: %w", err)
	}

	return authTokens{access: resp.Token, refresh: resp.RefreshToken}, nil
}

// getTokensByGithub dispatches to the configured GitHub login flow (device or browser).
// Default (empty) is "browser".
func getTokensByGithub(cli *client.Client, flow string, writer io.Writer, logger *slog.Logger) (authTokens, error) {
	switch flow {
	case "", config.GithubAuthFlowBrowser:
		return getTokensByGithubBrowser(cli, writer, logger)
	case config.GithubAuthFlowDevice:
		return getTokensByGithubDevice(cli, writer)
	default:
		return authTokens{}, &UnsupportedAuthMethodError{Method: "github:" + flow}
	}
}

// getTokensByGithubDevice implements the OAuth2 device authorization grant.
func getTokensByGithubDevice(cli *client.Client, writer io.Writer) (authTokens, error) {
	deviceAuthResponse, err := cli.AuthService.GetDeviceAuthToken()
	if err != nil {
		return authTokens{}, fmt.Errorf("failed to get device auth token: %w", err)
	}

	_, err = fmt.Fprintf(writer,
		"Please open the following URL in your browser: %s\n"+
			"Then enter the user code: %s\n"+
			"Wait for authentication...\n",
		deviceAuthResponse.VerificationURI,
		deviceAuthResponse.UserCode,
	)
	if err != nil {
		return authTokens{}, fmt.Errorf("failed to write device auth instructions: %w", err)
	}

	resp, err := cli.AuthService.ExchangeDeviceAuthToken(
		deviceAuthResponse.DeviceCode,
		deviceAuthResponse.Expiry.Time,
	)
	if err != nil {
		return authTokens{}, fmt.Errorf("failed to exchange device auth token: %w", err)
	}

	return authTokens{access: resp.Token, refresh: resp.RefreshToken}, nil
}

// persistTokens writes the access and refresh tokens to the file cache.
// Failures are logged but not propagated, so login still succeeds even if caching fails.
func persistTokens(cache *filecache.FileCache, logger *slog.Logger, access, refresh string) {
	if access != "" {
		err := cache.Set(BearerTokenKey, []byte(access))
		if err != nil {
			logger.Warn("failed to cache bearer token", "error", err)
		}
	}

	if refresh != "" {
		err := cache.Set(RefreshTokenKey, []byte(refresh))
		if err != nil {
			logger.Warn("failed to cache refresh token", "error", err)
		}
	}
}

// TokenPath constructs the file path for the cached token of a specific user.
func TokenPath(username string) string {
	return filepath.Join(username, TokenPrefix)
}
