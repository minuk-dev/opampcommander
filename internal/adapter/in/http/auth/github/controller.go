// Package github provides the GitHub oauth2 authentication controller for the opampcommander.
package github

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	v1auth "github.com/minuk-dev/opampcommander/api/v1/auth"
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
			Path:        "/api/v1/auth/github/device",
			Handler:     "http.github.GetDeviceAuth",
			HandlerFunc: c.GetDeviceAuth,
		},
		{
			Method:      "GET",
			Path:        "/api/v1/auth/github/device/exchange",
			Handler:     "http.github.ExchangeDeviceAuth",
			HandlerFunc: c.ExchangeDeviceAuth,
		},
	}
}

// HTTPAuth handles the HTTP request for GitHub OAuth2 authentication.
//
// @Summary  GitHub OAuth2 Authentication
// @Tags auth, github
// @Description Redirects to GitHub for OAuth2 authentication.
// @Accept   json
// @Produce  json
// @Success  302
// @Failure  500 {object} map[string]any
// @Router   /auth/github [get].
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
//
// @Summary  GitHub OAuth2 Authentication
// @Tags auth, github
// @Description Returns the GitHub OAuth2 authentication URL.
// @Accept json
// @Produce json
// @Success 200 {object} OAuth2AuthCodeURLResponse
// @Failure 500 {object} map[string]any
// @Router  /api/v1/auth/github [get].
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
	ctx.JSON(http.StatusOK, v1auth.OAuth2AuthCodeURLResponse{
		URL: authcodeURL,
	})
}

// Callback handles the callback from GitHub after the user has authenticated.
//
// @Summary	GitHub OAuth2 Callback
// @Tags auth, github
// @Description Exchanges the code received from GitHub for an authentication token.
// @Accept json
// @Produce json
// @Param state query string true "State parameter to prevent CSRF attacks"
// @Param code query string true "Code received from GitHub after authentication"
// @Success 200 {object} AuthnTokenResponse
// @Failure 500 {object} map[string]any
// @Router /auth/github/callback [get].
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

	ctx.JSON(http.StatusOK, v1auth.AuthnTokenResponse{
		Token: token,
	})
}

// GetDeviceAuth handles the request to get device authentication information.
//
// @Summary GitHub Device Authentication
// @Tags auth, github
// @Description Initiates device authorization for GitHub OAuth2.
// @Accept json
// @Produce json
// @Success 200 {object} DeviceAuthnTokenResponse
// @Failure 500 {object} map[string]any
// @Router /api/v1/auth/github/device [get].
func (c *Controller) GetDeviceAuth(ctx *gin.Context) {
	dar, err := c.service.DeviceAuth(ctx.Request.Context())
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to initiate device authorization",
			"details": fmt.Sprintf("error: %v", err),
		})

		return
	}

	ctx.JSON(http.StatusOK, v1auth.DeviceAuthnTokenResponse{
		DeviceCode:              dar.DeviceCode,
		UserCode:                dar.UserCode,
		VerificationURI:         dar.VerificationURI,
		VerificationURIComplete: dar.VerificationURIComplete,
		Expiry:                  dar.Expiry,
		Interval:                dar.Interval,
	})
}

// ExchangeDeviceAuth handles the request to exchange a device code for an authentication token.
// It expects the request to contain a device code and an optional expiry time.
//
// @Summary  GitHub Device Code Exchange
// @Tags auth, github
// @Description Exchanges a device code for an authentication token.
// @Accept json
// @Produce json
// @Param device_code query string true "Device code to exchange"
// @Param expiry  query string false "Optional expiry time in RFC3339 format"
// @Success 200 {object} AuthnTokenResponse
// @Failure 400 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/auth/github/device/exchange [get].
func (c *Controller) ExchangeDeviceAuth(ctx *gin.Context) {
	deviceCode := ctx.Query("device_code")
	expiry := ctx.Query("expiry")

	var expiryTime time.Time

	var err error
	if expiry != "" {
		expiryTime, err = time.Parse(time.RFC3339, expiry)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid expiry format",
				"details": fmt.Sprintf("error: %v", err),
			})

			return
		}
	}

	token, err := c.service.ExchangeDeviceAuth(
		ctx.Request.Context(),
		deviceCode,
		expiryTime,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to exchange device code for token",
			"details": fmt.Sprintf("error: %v", err),
		})

		return
	}

	ctx.JSON(http.StatusOK, v1auth.AuthnTokenResponse{
		Token: token,
	})
}
