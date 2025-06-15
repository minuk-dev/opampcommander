// Package basic provides a basic authentication controller for the opampcommander API client.
package basic

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	v1auth "github.com/minuk-dev/opampcommander/api/v1/auth"
	"github.com/minuk-dev/opampcommander/internal/security"
)

// Controller is a struct that implements the basic authentication controller for the opampcommander API client.
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

// RoutesInfo returns the routes information for the basic authentication controller.
func (c *Controller) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
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
//
// @Summary Basic Authentication
// @Tags auth, basic
// @Description Authenticate using basic auth credentials.
// @Accept json
// @Produce json
// @Success 200 {object} v1auth.AuthnTokenResponse
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

	token, err := c.service.BasicAuth(username, password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to authenticate",
			"details": fmt.Sprintf("error: %v", err),
		})

		return
	}

	ctx.JSON(http.StatusOK, v1auth.AuthnTokenResponse{
		Token: token,
	})
}
