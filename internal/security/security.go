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
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/oauth2"
	oauth2github "golang.org/x/oauth2/github"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
)

// Usecase defines the use case for the security package.
type Usecase interface {
	// ValidateToken validates the provided JWT token string and returns the claims if valid.
	ValidateToken(tokenString string) (*OPAMPClaims, error)

	// AdminUsecase returns the use case for admin authentication.
	AdminUsecase
	// OAuth2Usecase returns the use case for OAuth2 authentication.
	OAuth2Usecase
}

// AdminUsecase defines the use case for admin authentication.
type AdminUsecase interface {
	// BasicAuth authenticates the user using basic authentication with username and password.
	BasicAuth(username, password string) (string, error)
}

// OAuth2Usecase defines the use case for OAuth2 authentication.
type OAuth2Usecase interface {
	// AuthCodeURL generates the OAuth2 authorization URL with a unique state parameter.
	AuthCodeURL() (string, error)
	// Exchange exchanges the OAuth2 authorization code for an access token.
	Exchange(ctx context.Context, state, code string) (string, error)
}

var _ Usecase = (*Service)(nil)

// Service provides security-related functionality for the opampcommander application.
type Service struct {
	logger             *slog.Logger
	oauth2Config       *oauth2.Config
	oauthStateSettings config.JWTSettings
	adminSettings      config.AdminSettings
	tokenSettings      config.JWTSettings
	tracerProvider     trace.TracerProvider
	httpClient         *http.Client
}

// OPAMPClaims defines the custom claims for the JWT token used for opampcommander's authentication.
// It includes the user's email and standard JWT registered claims.
type OPAMPClaims struct {
	jwt.RegisteredClaims

	Email string `json:"email"`
}

// New creates a new instance of the Service struct with the provided logger and OAuth settings.
func New(
	logger *slog.Logger,
	settings *config.AuthSettings,
	tracerProvider trace.TracerProvider,
) *Service {
	// Create an HTTP client with OpenTelemetry instrumentation for tracing OAuth calls
	var httpClient *http.Client
	if tracerProvider != nil {
		httpClient = &http.Client{
			Transport: otelhttp.NewTransport(
				http.DefaultTransport,
				otelhttp.WithTracerProvider(tracerProvider),
			),
		}
	} else {
		httpClient = http.DefaultClient
	}

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
		tracerProvider:     tracerProvider,
		httpClient:         httpClient,
	}
}

// ValidateToken validates the provided JWT token string and returns the claims if valid.
// It checks the token's validity, expiration, and the email in the claims.
func (s *Service) ValidateToken(tokenString string) (*OPAMPClaims, error) {
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
		return "", ErrInvalidUsernameOrPassword
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

	tokenString, err := token.SignedString([]byte(s.tokenSettings.SigningKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT token: %w", err)
	}

	s.logger.Debug("Created JWT token", slog.String("token", tokenString))

	return tokenString, nil
}

func closeSilently(closer io.Closer) {
	_ = closer.Close() // Ignore error
}
