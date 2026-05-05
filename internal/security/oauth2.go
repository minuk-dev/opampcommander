package security

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-github/v72/github"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
)

const (
	// StateLength defines the length of the state string to be generated for OAuth2 authentication.
	StateLength = 16 // Length of the state string to be generated for OAuth2 authentication.
)

// Exchange exchanges the OAuth2 authorization code for an access token.
// It validates the state parameter to prevent CSRF attacks.
func (s *Service) Exchange(ctx context.Context, state, code string) (LoginResult, error) {
	err := s.validateState(state)
	if err != nil {
		return LoginResult{}, fmt.Errorf("invalid state: %w", err)
	}

	// Use context with custom HTTP client for tracing
	ctx = context.WithValue(ctx, oauth2.HTTPClient, s.httpClient)

	token, err := s.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return LoginResult{}, fmt.Errorf("failed to exchange OAuth2 code for token: %w", err)
	}

	s.logger.Debug("Exchanged OAuth2 code for token", slog.String("token", token.AccessToken))

	tokenType := strings.ToLower(token.TokenType)
	if tokenType != "bearer" {
		return LoginResult{}, &UnsupportedTokenTypeError{TokenType: tokenType}
	}

	authClient := s.oauth2Config.Client(ctx, token)
	if authClient == nil {
		return LoginResult{}, ErrOAuth2ClientCreationFailed
	}

	client := github.NewClient(authClient)

	emails, resp, err := client.Users.ListEmails(ctx, nil)
	if err != nil {
		return LoginResult{}, fmt.Errorf("failed to list user emails: %w", err)
	}

	defer closeSilently(resp.Body)

	email, found := lo.Find(emails, func(email *github.UserEmail) bool {
		return email != nil && email.GetPrimary() && email.GetVerified() && email.GetEmail() != ""
	})
	if !found {
		return LoginResult{}, ErrNoPrimaryEmailFound
	}

	groups := s.listGitHubOrgs(ctx, client)

	result, err := s.issueLoginResult(email.GetEmail())
	if err != nil {
		return LoginResult{}, err
	}

	result.Groups = groups
	s.logger.Debug("Created JWT token for user", slog.String("email", result.Email))

	return result, nil
}

// DeviceAuth initiates the OAuth2 device authorization flow.
// It returns a device authorization response that contains the user code and verification URL.
func (s *Service) DeviceAuth(ctx context.Context) (*oauth2.DeviceAuthResponse, error) {
	// Use context with custom HTTP client for tracing
	ctx = context.WithValue(ctx, oauth2.HTTPClient, s.httpClient)

	deviceAuthRes, err := s.oauth2Config.DeviceAuth(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate device authorization: %w", err)
	}

	return deviceAuthRes, nil
}

// ExchangeDeviceAuth exchanges the device code for an access token.
// It retrieves the user's primary email from GitHub and creates a JWT token with the email as a claim.
// It returns the JWT token string or an error if the process fails.
func (s *Service) ExchangeDeviceAuth(
	ctx context.Context,
	deviceCode string,
	expiry time.Time,
) (LoginResult, error) {
	// Use context with custom HTTP client for tracing
	ctx = context.WithValue(ctx, oauth2.HTTPClient, s.httpClient)

	token, err := s.oauth2Config.DeviceAccessToken(ctx,
		//exhaustruct:ignore
		&oauth2.DeviceAuthResponse{
			DeviceCode: deviceCode,
			Expiry:     expiry,
		})
	if err != nil {
		return LoginResult{}, fmt.Errorf("failed to exchange device code for token: %w", err)
	}

	s.logger.Debug("Exchanged OAuth2 code for token", slog.String("token", token.AccessToken))

	tokenType := strings.ToLower(token.TokenType)
	if tokenType != "bearer" {
		return LoginResult{}, &UnsupportedTokenTypeError{TokenType: tokenType}
	}

	authClient := s.oauth2Config.Client(ctx, token)
	if authClient == nil {
		return LoginResult{}, ErrOAuth2ClientCreationFailed
	}

	client := github.NewClient(authClient)

	emails, resp, err := client.Users.ListEmails(ctx, nil)
	if err != nil {
		return LoginResult{}, fmt.Errorf("failed to list user emails: %w", err)
	}

	defer closeSilently(resp.Body)

	email, found := lo.Find(emails, func(email *github.UserEmail) bool {
		return email != nil && email.GetPrimary() && email.GetVerified() && email.GetEmail() != ""
	})
	if !found {
		return LoginResult{}, ErrNoPrimaryEmailFound
	}

	groups := s.listGitHubOrgs(ctx, client)

	result, err := s.issueLoginResult(email.GetEmail())
	if err != nil {
		return LoginResult{}, err
	}

	result.Groups = groups
	s.logger.Debug("Created JWT token for user", slog.String("email", result.Email))

	return result, nil
}

// listGitHubOrgs fetches the authenticated user's GitHub org memberships.
// Failures are logged and return nil so login is never blocked.
func (s *Service) listGitHubOrgs(ctx context.Context, client *github.Client) []string {
	orgs, _, err := client.Organizations.List(ctx, "", nil)
	if err != nil {
		s.logger.Warn("failed to list GitHub orgs, continuing without org info",
			slog.String("error", err.Error()),
		)

		return nil
	}

	groups := make([]string, 0, len(orgs))

	for _, org := range orgs {
		if org.GetLogin() != "" {
			groups = append(groups, org.GetLogin())
		}
	}

	return groups
}

func (s *Service) validateState(state string) error {
	_, err := s.parseStateClaims(state)

	return err
}

func (s *Service) parseStateClaims(state string) (*OAuthStateClaims, error) {
	token, err := jwt.ParseWithClaims(state,
		//exhaustruct:ignore
		&OAuthStateClaims{},
		func(_ *jwt.Token) (any, error) {
			return []byte(s.oauthStateSettings.SigningKey), nil
		})
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT token for state: %w", err)
	}

	if !token.Valid {
		return nil, ErrInvalidState
	}

	claims, ok := token.Claims.(*OAuthStateClaims)
	if !ok {
		return nil, ErrInvalidState
	}

	exp, err := claims.GetExpirationTime()
	if err != nil {
		return nil, fmt.Errorf("failed to get expiration time from token claims: %w", err)
	}

	if exp == nil || exp.Before(time.Now()) {
		return nil, ErrStateExpired
	}

	s.logger.Debug("Validated state token", slog.String("state", state))

	return claims, nil
}

// OAuthStateClaims defines the custom claims for the JWT token used for the state parameter in OAuth2 authentication.
type OAuthStateClaims struct {
	jwt.RegisteredClaims

	// CLIRedirect, when non-empty, signals the callback handler to redirect the browser
	// to this loopback URI (with token query params) instead of returning JSON.
	CLIRedirect string `json:"cliRedirect,omitempty"`
}

func (s *Service) createState(cliRedirect string) (string, error) {
	randBytes := make([]byte, StateLength)

	_, err := rand.Read(randBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes for state: %w", err)
	}

	now := time.Now()
	claims := OAuthStateClaims{
		CLIRedirect: cliRedirect,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.oauthStateSettings.Issuer,
			Subject:   "oauth2_state",
			Audience:  s.oauthStateSettings.Audience,
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.oauthStateSettings.Expiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        base64.URLEncoding.EncodeToString(randBytes),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	ss, err := token.SignedString([]byte(s.oauthStateSettings.SigningKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT token for state: %w", err)
	}

	return ss, nil
}
