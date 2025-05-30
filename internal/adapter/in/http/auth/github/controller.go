// Package github provides the GitHub oauth2 authentication controller for the opampcommander.
package github

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/minuk-dev/opampcommander/internal/adapter/in/http/v1/auth"
	"github.com/minuk-dev/opampcommander/internal/security"
)

// Controller is a struct that implements the GitHub OAuth2 authentication controller.
type Controller struct {
	logger  *slog.Logger
	service *security.Service
}

// NewController creates a new instance of the Controller struct with the provided settings.
func NewController(
	logger *slog.Logger,
	service *security.Service,
) *Controller {
	return &Controller{
		logger:  logger,
		service: service,
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
		{
			Method:      "GET",
			Path:        "/api/v1/auth/basic",
			Handler:     "http.github.BasicAuth",
			HandlerFunc: c.BasicAuth,
		},
	}
}

// BasicAuth handles the HTTP request for basic authentication.
// It expects the request to contain basic auth credentials in the format "username:password".
func (c *Controller) BasicAuth(ctx *gin.Context) {
	username, password, ok := ctx.Request.BasicAuth()
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "missing basic auth credentials",
		})

		return
	}

	token, err := c.service.BasicAuth(username, password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to authenticate",
			"details": fmt.Sprintf("error: %v", err),
		})

		return
	}

	ctx.JSON(http.StatusOK, auth.AuthnTokenResponse{
		Token: token,
	})
}

// HTTPAuth handles the HTTP request for GitHub OAuth2 authentication.
func (c *Controller) HTTPAuth(ctx *gin.Context) {
	authcodeURL, err := c.service.AuthCodeURL()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to generate state",
			"details": fmt.Sprintf("error: %v", err),
		})

		return
	}

	ctx.Redirect(http.StatusTemporaryRedirect, authcodeURL)
}

// APIAuth handles the API request for GitHub OAuth2 authentication.
func (c *Controller) APIAuth(ctx *gin.Context) {
	authcodeURL, err := c.service.AuthCodeURL()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to generate state",
			"details": fmt.Sprintf("error: %v", err),
		})

		return
	}

	c.logger.Info("Generated auth code URL", slog.String("auth_url", authcodeURL))
	ctx.JSON(http.StatusOK, auth.OAuth2AuthCodeURLResponse{
		URL: authcodeURL,
	})
}

// Callback handles the callback from GitHub after the user has authenticated.
func (c *Controller) Callback(ctx *gin.Context) {
	state := ctx.Query("state")
	code := ctx.Query("code")

	token, err := c.service.Exchange(ctx.Request.Context(), state, code)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to exchange code for token",
			"details": fmt.Sprintf("error: %v", err),
		})

		return
	}

	ctx.JSON(http.StatusOK, auth.AuthnTokenResponse{
		Token: token,
	})
}
