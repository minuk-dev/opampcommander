// Package github provides the GitHub oauth2 authentication controller for the opampcommander.
package github

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/minuk-dev/opampcommander/pkg/app/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

const (
	// StateLength defines the length of the state string to be generated for OAuth2 authentication.
	StateLength = 16 // Length of the state string to be generated for OAuth2 authentication.
)

// OAuthStateGeneratorSettings holds the settings for generating OAuth2 state.
type OAuthStateGeneratorSettings struct{}

// Controller is a struct that implements the GitHub OAuth2 authentication controller.
type Controller struct {
	logger                      *slog.Logger
	oauth2Config                *oauth2.Config
	oauthStateGeneratorSettings *OAuthStateGeneratorSettings
}

// NewController creates a new instance of the Controller struct with the provided settings.
func NewController(
	logger *slog.Logger,
	settings *config.OAuthSettings,
) *Controller {
	oauth2Config := &oauth2.Config{
		ClientID:     settings.ClientID,
		ClientSecret: settings.Secret,
		RedirectURL:  settings.CallbackURL,
		Scopes:       []string{"user:email"},
		Endpoint:     github.Endpoint,
	}

	return &Controller{
		logger:                      logger,
		oauth2Config:                oauth2Config,
		oauthStateGeneratorSettings: &OAuthStateGeneratorSettings{},
	}
}

// RoutesInfo returns the routes information for the GitHub OAuth2 authentication controller.
func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      "GET",
			Path:        "/auth/github",
			Handler:     "http.github.HTTPAuth",
			HandlerFunc: c.HTTPAuth,
		},
		{
			Method:      "GET",
			Path:        "/auth/github/callback",
			Handler:     "http.github.Callback",
			HandlerFunc: c.Callback,
		},
		{
			Method:      "GET",
			Path:        "/api/v1/auth/github",
			Handler:     "http.github.APIAuth",
			HandlerFunc: c.APIAuth,
		},
	}
}

// HTTPAuth handles the HTTP request for GitHub OAuth2 authentication.
func (c *Controller) HTTPAuth(ctx *gin.Context) {
	state, err := c.createState()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to generate state",
			"details": fmt.Sprintf("error: %v", err),
		})

		return
	}

	authcodeURL := c.oauth2Config.AuthCodeURL(state)
	c.logger.Info("Generated auth code URL", slog.String("auth_url", authcodeURL))
	ctx.Redirect(http.StatusTemporaryRedirect, authcodeURL)
}

// Callback handles the callback from GitHub after the user has authenticated.
func (c *Controller) Callback(ctx *gin.Context) {
	state := ctx.Query("state")

	err := c.validateState(state)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid state",
			"details": fmt.Sprintf("error: %v", err),
		})

		return
	}

	token, err := c.oauth2Config.Exchange(ctx.Request.Context(), ctx.Request.FormValue("code"))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to exchange code for token",
			"details": fmt.Sprintf("error: %v", err),
		})

		return
	}

	ctx.JSON(http.StatusOK, token)
}

// APIAuth handles the API request for GitHub OAuth2 authentication.
func (c *Controller) APIAuth(ctx *gin.Context) {
	state, err := c.createState()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to generate state",
			"details": fmt.Sprintf("error: %v", err),
		})

		return
	}

	authcodeURL := c.oauth2Config.AuthCodeURL(state)
	c.logger.Info("Generated auth code URL", slog.String("auth_url", authcodeURL))
	ctx.JSON(http.StatusOK, gin.H{
		"auth_url": authcodeURL,
	})
}

// TODO: Implement a proper state management system.
func (c *Controller) createState() (string, error) {
	randBytes := make([]byte, StateLength)

	_, err := rand.Read(randBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes for state: %w", err)
	}

	state := base64.URLEncoding.EncodeToString(randBytes)

	return state, nil
}

// TODO: Implement a proper state validation system.
func (c *Controller) validateState(state string) error {
	return nil
}
