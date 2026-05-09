// Package security provides security-related functionality for the opampcommander application.
package security

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	oauth2github "golang.org/x/oauth2/github"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
)

// Token type values stored in the OPAMPClaims.TokenType claim.
const (
	// TokenTypeAccess marks a JWT issued as an access token.
	TokenTypeAccess = "access"
	// TokenTypeRefresh marks a JWT issued as a refresh token, accepted only by the refresh endpoint.
	TokenTypeRefresh = "refresh"
)

// LoginResult contains the result of a successful authentication.
type LoginResult struct {
	Token        string
	RefreshToken string
	ExpiresAt    time.Time
	Email        string
	Groups       []string // provider group/org memberships, added as labels to the user on login
}

// Usecase defines the use case for the security package.
type Usecase interface {
	// ValidateToken validates the provided JWT token string and returns the claims if valid.
	ValidateToken(tokenString string) (*OPAMPClaims, error)
	// Refresh exchanges a valid refresh token for a new access token (and rotated refresh token).
	Refresh(refreshToken string) (LoginResult, error)

	// AdminUsecase returns the use case for admin authentication.
	AdminUsecase
	// OAuth2Usecase returns the use case for OAuth2 authentication.
	OAuth2Usecase
}

// AdminUsecase defines the use case for admin authentication.
type AdminUsecase interface {
	// BasicAuth authenticates the user using basic authentication with username and password.
	BasicAuth(username, password string) (LoginResult, error)
}

// OAuth2Usecase defines the use case for OAuth2 authentication.
type OAuth2Usecase interface {
	// AuthCodeURL generates the OAuth2 authorization URL with a unique state parameter.
	// cliRedirect is an optional loopback redirect URI (e.g. http://127.0.0.1:PORT/callback)
	// to be encoded into the state JWT; on callback the server redirects to it instead of
	// returning JSON. Empty string preserves the legacy JSON behavior.
	AuthCodeURL(cliRedirect string) (string, error)
	// CLIRedirectFromState extracts the loopback redirect URI from a state JWT, if any.
	CLIRedirectFromState(state string) (string, error)
	// Exchange exchanges the OAuth2 authorization code for an access token.
	Exchange(ctx context.Context, state, code string) (LoginResult, error)
}

var _ Usecase = (*Service)(nil)

// Service provides security-related functionality for the opampcommander application.
type Service struct {
	logger             *slog.Logger
	oauth2Config       *oauth2.Config
	oauthStateSettings config.JWTSettings
	adminSettings      config.AdminSettings
	tokenSettings      config.JWTSettings
	httpClient         *http.Client
}

// OPAMPClaims defines the custom claims for the JWT token used for opampcommander's authentication.
// It includes the user's email and standard JWT registered claims.
type OPAMPClaims struct {
	jwt.RegisteredClaims

	Email string `json:"email"`
	// TokenType is "access" or "refresh". Empty value is treated as "access" for backward compatibility.
	TokenType string `json:"tokenType,omitempty"`
}

// New creates a new instance of the Service struct with the provided logger and OAuth settings.
func New(
	logger *slog.Logger,
	settings *config.AuthSettings,
	httpClient *http.Client,
) *Service {
	var oauth2Cfg *oauth2.Config
	if settings.OAuthSettings != nil {
		oauth2Cfg = &oauth2.Config{
			ClientID:     settings.OAuthSettings.ClientID,
			ClientSecret: settings.OAuthSettings.Secret,
			RedirectURL:  settings.OAuthSettings.CallbackURL,
			Scopes:       []string{"user:email", "read:org"},
			Endpoint:     oauth2github.Endpoint,
		}
	}

	return &Service{
		logger:             logger,
		oauth2Config:       oauth2Cfg,
		oauthStateSettings: settings.JWTSettings,
		adminSettings:      settings.AdminSettings,
		tokenSettings:      settings.JWTSettings,
		httpClient:         httpClient,
	}
}

// ValidateToken validates the provided JWT token string and returns the claims if valid.
// It checks the token's validity, expiration, and rejects refresh tokens.
func (s *Service) ValidateToken(tokenString string) (*OPAMPClaims, error) {
	claims, err := s.parseClaims(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType == TokenTypeRefresh {
		return nil, ErrInvalidToken
	}

	s.logger.Debug("Validated JWT token", slog.String("email", claims.Email))

	return claims, nil
}

// Refresh validates a refresh token and issues a new access/refresh token pair for the same user.
// It rotates the refresh token (the old one remains valid until its own expiry — JWTs are stateless).
func (s *Service) Refresh(refreshToken string) (LoginResult, error) {
	claims, err := s.parseClaims(refreshToken)
	if err != nil {
		return LoginResult{}, err
	}

	if claims.TokenType != TokenTypeRefresh {
		return LoginResult{}, ErrInvalidToken
	}

	result, err := s.issueLoginResult(claims.Email)
	if err != nil {
		return LoginResult{}, fmt.Errorf("failed to issue tokens on refresh: %w", err)
	}

	s.logger.Debug("Refreshed tokens for user", slog.String("email", claims.Email))

	return result, nil
}

// AuthCodeURL generates the OAuth2 authorization URL with a unique state parameter.
// If cliRedirect is non-empty, it is encoded into the state JWT and used by the callback
// handler to redirect to a loopback URL instead of returning JSON.
func (s *Service) AuthCodeURL(cliRedirect string) (string, error) {
	state, err := s.createState(cliRedirect)
	if err != nil {
		return "", err
	}

	authURL := s.oauth2Config.AuthCodeURL(state)
	s.logger.Debug("Generated OAuth2 authorization URL", slog.String("url", authURL))

	return authURL, nil
}

// CLIRedirectFromState parses the state JWT and returns the embedded CLI loopback redirect, if any.
// Returns an empty string when the state has no CLI redirect.
func (s *Service) CLIRedirectFromState(state string) (string, error) {
	claims, err := s.parseStateClaims(state)
	if err != nil {
		return "", err
	}

	return claims.CLIRedirect, nil
}

// BasicAuth authenticates the user using basic authentication with username and password.
func (s *Service) BasicAuth(username, password string) (LoginResult, error) {
	if username != s.adminSettings.Username || password != s.adminSettings.Password {
		return LoginResult{}, ErrInvalidUsernameOrPassword
	}

	s.logger.Debug("Authenticated user with basic auth", slog.String("username", username))

	result, err := s.issueLoginResult(s.adminSettings.Email)
	if err != nil {
		return LoginResult{}, err
	}

	s.logger.Debug("Created JWT token for user", slog.String("email", result.Email))

	return result, nil
}

func (s *Service) parseClaims(tokenString string) (*OPAMPClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString,
		//exhaustruct:ignore
		&OPAMPClaims{},
		func(_ *jwt.Token) (any, error) {
			return []byte(s.tokenSettings.SigningKey), nil
		})
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT token: %w", err)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*OPAMPClaims)
	if !ok {
		return nil, ErrInvalidTokenClaims
	}

	exp, err := claims.GetExpirationTime()
	if err != nil {
		return nil, ErrTokenExpired
	}

	if exp == nil || exp.Before(time.Now()) {
		return nil, ErrStateExpired
	}

	return claims, nil
}

// issueLoginResult creates a fresh access (and optional refresh) token pair for the given email.
func (s *Service) issueLoginResult(email string) (LoginResult, error) {
	accessClaims := s.newOPAMPClaims(email, TokenTypeAccess, s.tokenSettings.Expiration)

	accessToken, err := s.createToken(accessClaims)
	if err != nil {
		return LoginResult{}, fmt.Errorf("failed to create access JWT: %w", err)
	}

	var refreshToken string

	if s.tokenSettings.RefreshExpiration > 0 {
		refreshClaims := s.newOPAMPClaims(email, TokenTypeRefresh, s.tokenSettings.RefreshExpiration)

		refreshToken, err = s.createToken(refreshClaims)
		if err != nil {
			return LoginResult{}, fmt.Errorf("failed to create refresh JWT: %w", err)
		}
	}

	return LoginResult{
		Token:        accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessClaims.ExpiresAt.Time,
		Email:        email,
		Groups:       nil,
	}, nil
}

func (s *Service) newOPAMPClaims(email, tokenType string, expiration time.Duration) *OPAMPClaims {
	now := time.Now()

	return &OPAMPClaims{
		Email:     email,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.tokenSettings.Issuer,
			Subject:   "opampcommander",
			Audience:  s.tokenSettings.Audience,
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        base64.URLEncoding.EncodeToString([]byte(email + ":" + tokenType + ":" + now.Format(time.RFC3339Nano))),
		},
	}
}

func (s *Service) createToken(claims *OPAMPClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(s.tokenSettings.SigningKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT token: %w", err)
	}

	s.logger.Debug("Created JWT token", slog.String("type", claims.TokenType))

	return tokenString, nil
}

func closeSilently(closer io.Closer) {
	_ = closer.Close() // Ignore error
}
