// Package basic provides a basic authentication controller for the opampcommander API client.
package basic

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	v1auth "github.com/minuk-dev/opampcommander/api/v1/auth"
	applicationport "github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/usecase"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/security"
)

// Controller is a struct that implements the basic authentication controller for the opampcommander API client.
type Controller struct {
	logger              *slog.Logger
	service             *security.Service
	provisioningUsecase usecase.AuthProvisioningUsecase
}

// NewController creates a new instance of the Controller struct with the provided settings.
func NewController(
	logger *slog.Logger,
	service *security.Service,
	provisioningUsecase usecase.AuthProvisioningUsecase,
) *Controller {
	return &Controller{
		logger:              logger,
		service:             service,
		provisioningUsecase: provisioningUsecase,
	}
}

// RoutesInfo returns the routes information for the basic authentication controller.
func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      "GET",
			Path:        "/api/v1/auth/basic",
			Handler:     "http.github.BasicAuth",
			HandlerFunc: c.BasicAuth,
		},
		{
			Method:      "GET",
			Path:        "/api/v1/auth/info",
			Handler:     "http.github.Info",
			HandlerFunc: c.Info,
		},
		{
			Method:      "POST",
			Path:        "/api/v1/auth/refresh",
			Handler:     "http.auth.Refresh",
			HandlerFunc: c.Refresh,
		},
	}
}

// BasicAuth handles the HTTP request for basic authentication.
// It expects the request to contain basic auth credentials in the format "username:password".
//
// @Summary Basic Authentication
// @Tags auth, basic
// @Description Authenticate using basic auth credentials.
// @Accept json
// @Produce json
// @Success 200 {object} AuthnTokenResponse
// @Failure 401 {object} map[string]any
// @Router /api/v1/auth/basic [get].
func (c *Controller) BasicAuth(ctx *gin.Context) {
	username, password, ok := ctx.Request.BasicAuth()
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "missing basic auth credentials",
		})

		return
	}

	result, err := c.service.BasicAuth(ctx.Request.Context(), username, password)
	if err != nil {
		if errors.Is(err, security.ErrInvalidUsernameOrPassword) {
			ctx.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid username or password",
			})

			return
		}

		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to authenticate",
			"details": fmt.Sprintf("error: %v", err),
		})

		return
	}

	c.provisioningUsecase.EnsureUserOnLogin(ctx.Request.Context(), applicationport.LoginProvisioning{
		Provider: applicationport.IdentityProviderBasic,
		Username: username,
		Email:    result.Email,
		Groups:   nil,
	})

	ctx.JSON(http.StatusOK, v1auth.AuthnTokenResponse{
		Token:        result.Token,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    v1.NewTime(result.ExpiresAt),
	})
}

// Refresh handles refresh token exchange.
//
// @Summary Refresh access token
// @Tags auth
// @Description Exchange a refresh token for a new access token (and rotated refresh token).
// @Accept json
// @Produce json
// @Param request body RefreshTokenRequest true "Refresh token"
// @Success 200 {object} AuthnTokenResponse
// @Failure 400 {object} map[string]any
// @Failure 401 {object} map[string]any
// @Router /api/v1/auth/refresh [post].
func (c *Controller) Refresh(ctx *gin.Context) {
	var req v1auth.RefreshTokenRequest

	err := ctx.ShouldBindJSON(&req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": fmt.Sprintf("error: %v", err),
		})

		return
	}

	result, err := c.service.Refresh(req.RefreshToken)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error":   "invalid or expired refresh token",
			"details": fmt.Sprintf("error: %v", err),
		})

		return
	}

	ctx.JSON(http.StatusOK, v1auth.AuthnTokenResponse{
		Token:        result.Token,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    v1.NewTime(result.ExpiresAt),
	})
}

// Info handles the HTTP request to get auth info.
//
// @Summary Info
// @Tags auth, basic
// @Description Get Authentication Info.
// @Produce json
// @Success 200 {object} InfoResponse
// @Failure 401 {object} map[string]any
// @Router /api/v1/auth/info [get].
func (c *Controller) Info(ctx *gin.Context) {
	user, err := security.GetUser(ctx)
	if err != nil {
		c.logger.Error("failed to get user from context",
			slog.String("ip", ctx.ClientIP()),
			slog.String("error", err.Error()),
		)
		ctx.JSON(http.StatusUnauthorized, gin.H{
			"error": "unauthorized",
		})

		return
	}

	ctx.JSON(http.StatusOK, v1auth.InfoResponse{
		Authenticated: user.Authenticated,
		Email:         user.Email,
	})
}
