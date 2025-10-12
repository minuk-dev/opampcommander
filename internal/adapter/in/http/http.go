// Package http provides HTTP adapter for the application.
// This package's structure follows the http path.
// e.g.
// - /api/v1/agents -> internal/adapter/in/http/v1/agents
// - /auth -> internal/adapter/in/http/auth
package http

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthService defines the interface for health checking services.
type HealthService interface {
	IsReady(ctx context.Context) bool
	IsHealth(ctx context.Context) bool
}

// HealthController handles health check HTTP requests.
type HealthController struct {
	healthService HealthService
}

// NewHealthController creates a new instance of HealthController.
func NewHealthController(healthService HealthService) *HealthController {
	return &HealthController{
		healthService: healthService,
	}
}

// RoutesInfo implements helper.Controller.
func (c *HealthController) RoutesInfo() gin.RoutesInfo {
	return gin.RoutesInfo{
		{
			Method:      http.MethodGet,
			Path:        "/healthz",
			Handler:     "http.HealthController.IsHealth",
			HandlerFunc: c.IsHealth,
		},
		{
			Method:      http.MethodGet,
			Path:        "/readyz",
			Handler:     "http.HealthController.IsReady",
			HandlerFunc: c.IsReady,
		},
	}
}

// IsReady checks if the service is ready.
//
// @Summary Readiness Check
// @Description Check if the service is ready
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {string} string "OK"
// @Failure 503 {string} string "Service Unavailable"
// @Router /readyz [get].
func (c *HealthController) IsReady(gCtx *gin.Context) {
	ctx := gCtx.Request.Context()
	if c.healthService.IsReady(ctx) {
		gCtx.Status(http.StatusOK)
	} else {
		gCtx.Status(http.StatusServiceUnavailable)
	}
}

// IsHealth checks if the service is healthy.
//
// @Summary Liveness Check
// @Description Check if the service is healthy
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {string} string "OK"
// @Failure 503 {string} string "Service Unavailable"
// @Router /healthz [get].
func (c *HealthController) IsHealth(gCtx *gin.Context) {
	ctx := gCtx.Request.Context()
	if c.healthService.IsHealth(ctx) {
		gCtx.Status(http.StatusOK)
	} else {
		gCtx.Status(http.StatusServiceUnavailable)
	}
}
