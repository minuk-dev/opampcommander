// Package security provides security-related functionality for the opampcommander application.
package security

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/minuk-dev/opampcommander/pkg/app/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

const (
	// StateLength defines the length of the state string to be generated for OAuth2 authentication.
	StateLength = 16 // Length of the state string to be generated for OAuth2 authentication.
)

// Service provides security-related functionality for the opampcommander application.
type Service struct {
	logger          *slog.Logger
	oauth2Config    *oauth2.Config
	stateSigningKey string
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
		stateSigningKey: settings.SigningKey,
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

type CustomClaims struct {
	foo string `json:"foo"`
	jwt.RegisteredClaims
}

// TODO: Implement a proper state management system.
func (c *Service) createState() (string, error) {
	randBytes := make([]byte, StateLength)

	_, err := rand.Read(randBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes for state: %w", err)
	}

	claims := CustomClaims{
		foo: "bar", // Example custom claim, can be removed or modified as needed
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "opampcommander",
			Subject:   "oauth2_state",
			Audience:  jwt.ClaimStrings{"opampcommander"},
			NotBefore: jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * 60 * 1000)), // 5 minutes expiration
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        base64.URLEncoding.EncodeToString(randBytes),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString([]byte(c.stateSigningKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT token for state: %w", err)
	}

	return ss, nil
}

// TODO: Implement a proper state validation system.
func (c *Service) validateState(state string) error {
	token, err := jwt.ParseWithClaims(state, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(c.stateSigningKey), nil
	})
	if err != nil {
		return fmt.Errorf("failed to parse JWT token for state: %w", err)
	}

	exp, err := token.Claims.GetExpirationTime()
	if err != nil {
		return fmt.Errorf("failed to get expiration time from token claims: %w", err)
	}

	if exp == nil || exp.Time.Before(time.Now()) {
		return fmt.Errorf("state token has expired")
	}
	if !token.Valid {
		return fmt.Errorf("state token is invalid")
	}

	c.logger.Debug("Validated state token", slog.String("state", state))
	return nil
}
