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
)

const (
	// StateLength defines the length of the state string to be generated for OAuth2 authentication.
	StateLength = 16 // Length of the state string to be generated for OAuth2 authentication.
)

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
		return "", &UnsupportedTokenTypeError{TokenType: tokenType}
	}

	authClient := s.oauth2Config.Client(ctx, token)
	if authClient == nil {
		return "", ErrOAuth2ClientCreationFailed
	}

	client := github.NewClient(authClient)

	emails, resp, err := client.Users.ListEmails(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to list user emails: %w", err)
	}

	defer closeSilently(resp.Body)

	email, found := lo.Find(emails, func(email *github.UserEmail) bool {
		return email != nil && email.GetPrimary() && email.GetVerified() && email.GetEmail() != ""
	})
	if !found {
		return "", ErrNoPrimaryEmailFound
	}

	claims := s.newOPAMPClaims(email.GetEmail())

	tokenString, err := s.createToken(claims)
	if err != nil {
		return "", fmt.Errorf("failed to create JWT token: %w", err)
	}

	s.logger.Debug("Created JWT token for user", slog.String("email", claims.Email))

	return tokenString, nil
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
