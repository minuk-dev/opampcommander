// Package security provides security-related functionality for the opampcommander application.
package security

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

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
	logger       *slog.Logger
	oauth2Config *oauth2.Config
	jwtSettings  config.JWTSettings
}

// New creates a new instance of the Service struct with the provided logger and OAuth settings.
func New(
	logger *slog.Logger,
	settings *config.OAuthSettings,
) *Service {
	return &Service{
		logger: logger,
		oauth2Config: &oauth2.Config{
			ClientID:     settings.ClientID,
			ClientSecret: settings.Secret,
			RedirectURL:  settings.CallbackURL,
			Scopes:       []string{"user:email"},
			Endpoint:     github.Endpoint,
		},
		jwtSettings: settings.JWTSettings,
	}
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

// Exchange exchanges the OAuth2 authorization code for an access token.
// It validates the state parameter to prevent CSRF attacks.
func (s *Service) Exchange(ctx context.Context, state, code string) (*oauth2.Token, error) {
	err := s.validateState(state)
	if err != nil {
		return nil, fmt.Errorf("invalid state: %w", err)
	}

	token, err := s.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange OAuth2 code for token: %w", err)
	}

	s.logger.Debug("Exchanged OAuth2 code for token", slog.String("token", token.AccessToken))

	return token, nil
}

// CustomClaims defines the custom claims for the JWT token used for the state parameter in OAuth2 authentication.
type CustomClaims struct {
	jwt.RegisteredClaims
}

func (s *Service) createState() (string, error) {
	randBytes := make([]byte, StateLength)

	_, err := rand.Read(randBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes for state: %w", err)
	}

	now := time.Now()
	claims := CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.jwtSettings.Issuer,
			Subject:   "oauth2_state",
			Audience:  s.jwtSettings.Audience,
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.jwtSettings.Expiration)), // 5 minutes expiration
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        base64.URLEncoding.EncodeToString(randBytes),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	ss, err := token.SignedString([]byte(s.jwtSettings.SigningKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT token for state: %w", err)
	}

	return ss, nil
}

func (s *Service) validateState(state string) error {
	token, err := jwt.ParseWithClaims(state,
		//exhaustruct:ignore
		&CustomClaims{}, // only to convey the type of claims we expect
		func(_ *jwt.Token) (interface{}, error) {
			return []byte(s.jwtSettings.SigningKey), nil
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
