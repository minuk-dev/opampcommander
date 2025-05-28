// Package security provides security-related functionality for the opampcommander application.
package security

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-github/v72/github"
	"github.com/samber/lo"
	"golang.org/x/oauth2"
	oauth2github "golang.org/x/oauth2/github"

	"github.com/minuk-dev/opampcommander/pkg/app/config"
)

const (
	// StateLength defines the length of the state string to be generated for OAuth2 authentication.
	StateLength = 16 // Length of the state string to be generated for OAuth2 authentication.
)

var (
	// ErrInvalidState is returned when the state parameter is invalid.
	ErrInvalidState = errors.New("invalid state parameter")
	// ErrStateExpired is returned when the state parameter has expired.
	ErrStateExpired = errors.New("state parameter has expired")
)

// Service provides security-related functionality for the opampcommander application.
type Service struct {
	logger             *slog.Logger
	oauth2Config       *oauth2.Config
	oauthStateSettings config.JWTSettings
	adminSettings      config.AdminSettings
	tokenSettings      config.JWTSettings
}

// New creates a new instance of the Service struct with the provided logger and OAuth settings.
func New(
	logger *slog.Logger,
	settings *config.AuthSettings,
) *Service {
	return &Service{
		logger: logger,
		oauth2Config: &oauth2.Config{
			ClientID:     settings.OAuthSettings.ClientID,
			ClientSecret: settings.OAuthSettings.Secret,
			RedirectURL:  settings.OAuthSettings.CallbackURL,
			Scopes:       []string{"user:email"},
			Endpoint:     oauth2github.Endpoint,
		},
		oauthStateSettings: settings.JWTSettings,
		adminSettings:      settings.AdminSettings,
		tokenSettings:      settings.JWTSettings,
	}
}

func (s *Service) ValidateToken(tokenString string) (*OPAMPClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString,
		&OPAMPClaims{},
		func(_ *jwt.Token) (interface{}, error) {
			return []byte(s.tokenSettings.SigningKey), nil
		})
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(*OPAMPClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	exp, err := claims.GetExpirationTime()
	if err != nil {
		return nil, fmt.Errorf("failed to get expiration time from token claims: %w", err)
	}

	if exp == nil || exp.Before(time.Now()) {
		return nil, ErrStateExpired
	}

	s.logger.Debug("Validated JWT token", slog.String("email", claims.Email))

	return claims, nil
}

// AuthCodeURL generates the OAuth2 authorization URL with a unique state parameter.
func (s *Service) AuthCodeURL() (string, error) {
	state, err := s.createState()
	if err != nil {
		return "", err
	}

	authURL := s.oauth2Config.AuthCodeURL(state)
	s.logger.Debug("Generated OAuth2 authorization URL", slog.String("url", authURL))

	return authURL, nil
}

// BasicAuth authenticates the user using basic authentication with username and password.
func (s *Service) BasicAuth(username, password string) (string, error) {
	if username != s.adminSettings.Username || password != s.adminSettings.Password {
		return "", errors.New("invalid username or password")
	}

	s.logger.Debug("Authenticated user with basic auth", slog.String("username", username))
	claims := s.newOPAMPClaims(username)

	tokenString, err := s.createToken(claims)
	if err != nil {
		return "", fmt.Errorf("failed to create JWT token: %w", err)
	}

	s.logger.Debug("Created JWT token for user", slog.String("email", claims.Email))

	return tokenString, nil
}

// Exchange exchanges the OAuth2 authorization code for an access token.
// It validates the state parameter to prevent CSRF attacks.
func (s *Service) Exchange(ctx context.Context, state, code string) (string, error) {
	err := s.validateState(state)
	if err != nil {
		return "", fmt.Errorf("invalid state: %w", err)
	}

	token, err := s.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return "", fmt.Errorf("failed to exchange OAuth2 code for token: %w", err)
	}

	s.logger.Debug("Exchanged OAuth2 code for token", slog.String("token", token.AccessToken))

	tokenType := strings.ToLower(token.TokenType)
	if tokenType != "bearer" {
		return "", fmt.Errorf("unsupported token type: %s", tokenType)
	}

	authClient := s.oauth2Config.Client(ctx, token)
	if authClient == nil {
		return "", errors.New("failed to create OAuth2 client")
	}

	client := github.NewClient(authClient)

	emails, resp, err := client.Users.ListEmails(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to list user emails: %w", err)
	}

	defer resp.Body.Close()

	email, found := lo.Find(emails, func(email *github.UserEmail) bool {
		return email != nil && email.GetPrimary() && email.GetVerified() && email.GetEmail() != ""
	})
	if !found {
		return "", errors.New("no primary verified email found")
	}

	claims := s.newOPAMPClaims(email.GetEmail())

	tokenString, err := s.createToken(claims)
	if err != nil {
		return "", fmt.Errorf("failed to create JWT token: %w", err)
	}

	s.logger.Debug("Created JWT token for user", slog.String("email", claims.Email))

	return tokenString, nil
}

type OPAMPClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func (s *Service) newOPAMPClaims(email string) *OPAMPClaims {
	now := time.Now()

	return &OPAMPClaims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.oauthStateSettings.Issuer,
			Subject:   "opampcommander",
			Audience:  s.oauthStateSettings.Audience,
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.oauthStateSettings.Expiration)), // 5 minutes expiration
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        base64.URLEncoding.EncodeToString([]byte(email)),
		},
	}
}

func (s *Service) createToken(claims *OPAMPClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	ss, err := token.SignedString([]byte(s.tokenSettings.SigningKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT token: %w", err)
	}

	s.logger.Debug("Created JWT token", slog.String("token", ss))

	return ss, nil
}

// OAuthStateClaims defines the custom claims for the JWT token used for the state parameter in OAuth2 authentication.
type OAuthStateClaims struct {
	jwt.RegisteredClaims
}

func (s *Service) createState() (string, error) {
	randBytes := make([]byte, StateLength)

	_, err := rand.Read(randBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes for state: %w", err)
	}

	now := time.Now()
	claims := OAuthStateClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.oauthStateSettings.Issuer,
			Subject:   "oauth2_state",
			Audience:  s.oauthStateSettings.Audience,
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.oauthStateSettings.Expiration)), // 5 minutes expiration
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

func (s *Service) validateState(state string) error {
	token, err := jwt.ParseWithClaims(state,
		//exhaustruct:ignore
		&OAuthStateClaims{}, // only to convey the type of claims we expect
		func(_ *jwt.Token) (interface{}, error) {
			return []byte(s.oauthStateSettings.SigningKey), nil
		})
	if err != nil {
		return fmt.Errorf("failed to parse JWT token for state: %w", err)
	}

	exp, err := token.Claims.GetExpirationTime()
	if err != nil {
		return fmt.Errorf("failed to get expiration time from token claims: %w", err)
	}

	if exp == nil || exp.Before(time.Now()) {
		return ErrStateExpired
	}

	if !token.Valid {
		return ErrInvalidState
	}

	s.logger.Debug("Validated state token", slog.String("state", state))

	return nil
}
