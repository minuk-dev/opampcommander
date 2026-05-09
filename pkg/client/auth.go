package client

import (
	"context"
	"fmt"
	"time"

	v1auth "github.com/minuk-dev/opampcommander/api/v1/auth"
)

const (
	// GithubAuthDeviceAuthAPIURL is the API URL for GitHub device authentication.
	GithubAuthDeviceAuthAPIURL = "/api/v1/auth/github/device"
	// GithubAuthExchangeDeviceAuthAPIURL is the API URL for exchanging GitHub device authentication tokens.
	GithubAuthExchangeDeviceAuthAPIURL = "/api/v1/auth/github/device/exchange"
	// GithubAuthCodeURLAPIURL is the API URL to obtain an auth-code URL bound to a CLI loopback redirect.
	GithubAuthCodeURLAPIURL = "/api/v1/auth/github/authcode"
	// BasicAuthAPIURL is the API URL for basic authentication.
	BasicAuthAPIURL = "/api/v1/auth/basic"
	// InfoAPIURL is the API URL to fetch auth info.
	InfoAPIURL = "/api/v1/auth/info"
	// RefreshAPIURL is the API URL for refreshing an access token.
	RefreshAPIURL = "/api/v1/auth/refresh"
)

// AuthService provides methods to interact with authentication resources.
type AuthService struct {
	service *service
}

// NewAuthService creates a new AuthService.
func NewAuthService(service *service) *AuthService {
	return &AuthService{
		service: service,
	}
}

// GetInfo retrieves authentication information from the server.
func (s *AuthService) GetInfo() (*v1auth.InfoResponse, error) {
	var authInfo v1auth.InfoResponse

	res, err := s.service.Resty.R().
		SetResult(&authInfo).
		Get(InfoAPIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth info: %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to get auth info: %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	if res.Result() == nil {
		return nil, fmt.Errorf("failed to get auth info: %w", ErrEmptyResponse)
	}

	return &authInfo, nil
}

// GetAuthTokenByBasicAuth retrieves an authentication token using basic authentication.
func (s *AuthService) GetAuthTokenByBasicAuth(username, password string) (*v1auth.AuthnTokenResponse, error) {
	var authToken v1auth.AuthnTokenResponse

	res, err := s.service.Resty.R().
		SetResult(&authToken).
		SetBasicAuth(username, password).
		Get(BasicAuthAPIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth token by basic auth: %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to get auth token by basic auth: %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	if res.Result() == nil {
		return nil, fmt.Errorf("failed to get auth token by basic auth: %w", ErrEmptyResponse)
	}

	return &authToken, nil
}

// GetDeviceAuthToken retrieves a device authentication token from GitHub.
func (s *AuthService) GetDeviceAuthToken() (*v1auth.DeviceAuthnTokenResponse, error) {
	var deviceAuthToken v1auth.DeviceAuthnTokenResponse

	res, err := s.service.Resty.R().
		SetResult(&deviceAuthToken).
		Get(GithubAuthDeviceAuthAPIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth token by github: %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to get auth token by github: %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	if res.Result() == nil {
		return nil, fmt.Errorf("failed to get auth token by github: %w", ErrEmptyResponse)
	}

	return &deviceAuthToken, nil
}

// ExchangeDeviceAuthToken exchanges a device code for an authentication token.
func (s *AuthService) ExchangeDeviceAuthToken(deviceCode string, expiry time.Time) (*v1auth.AuthnTokenResponse, error) {
	var authToken v1auth.AuthnTokenResponse

	// This request blocks on the server while it polls GitHub for authorization.
	// The shared Resty client has a 15s hard timeout which is too short for interactive login.
	// Clone the client and clear the timeout; the context deadline is the only limit.
	exchangeClient := s.service.Resty.Clone().SetTimeout(0)

	req := exchangeClient.R().
		SetResult(&authToken).
		SetQueryParam("device_code", deviceCode)

	if !expiry.IsZero() {
		req = req.SetQueryParam("expiry", expiry.Format(time.RFC3339))

		ctx, cancel := context.WithDeadline(context.Background(), expiry)
		defer cancel()

		req = req.SetContext(ctx)
	}

	res, err := req.Get(GithubAuthExchangeDeviceAuthAPIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange device auth token: %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to exchange device auth token: %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	if res.Result() == nil {
		return nil, fmt.Errorf("failed to exchange device auth token: %w", ErrEmptyResponse)
	}

	return &authToken, nil
}

// GetAuthCodeURL retrieves a GitHub OAuth2 authorization URL bound to the given loopback redirect URI.
// The server encodes the redirect URI into the state JWT; on callback the browser is redirected
// to the loopback URI with the tokens as query parameters.
func (s *AuthService) GetAuthCodeURL(redirectURI string) (*v1auth.OAuth2AuthCodeURLResponse, error) {
	var resp v1auth.OAuth2AuthCodeURLResponse

	res, err := s.service.Resty.R().
		SetResult(&resp).
		SetQueryParam("redirect_uri", redirectURI).
		Get(GithubAuthCodeURLAPIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth code URL: %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to get auth code URL: %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	return &resp, nil
}

// Refresh exchanges a refresh token for a new access (and rotated refresh) token.
func (s *AuthService) Refresh(refreshToken string) (*v1auth.AuthnTokenResponse, error) {
	var authToken v1auth.AuthnTokenResponse

	res, err := s.service.Resty.R().
		SetResult(&authToken).
		SetBody(v1auth.RefreshTokenRequest{RefreshToken: refreshToken}).
		Post(RefreshAPIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	if res.IsError() {
		return nil, fmt.Errorf("failed to refresh token: %w", &ResponseError{
			StatusCode:   res.StatusCode(),
			ErrorMessage: res.String(),
		})
	}

	if res.Result() == nil {
		return nil, fmt.Errorf("failed to refresh token: %w", ErrEmptyResponse)
	}

	return &authToken, nil
}
