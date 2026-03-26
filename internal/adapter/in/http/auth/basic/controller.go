// Package basic provides a basic authentication controller for the opampcommander API client.
package basic

import (
"context"
"errors"
"fmt"
"log/slog"
"net/http"
"time"

"github.com/gin-gonic/gin"

v1auth "github.com/minuk-dev/opampcommander/api/v1/auth"
"github.com/minuk-dev/opampcommander/internal/domain/model"
domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
"github.com/minuk-dev/opampcommander/internal/security"
)

// Controller is a struct that implements the basic authentication controller for the opampcommander API client.
type Controller struct {
logger      *slog.Logger
service     *security.Service
userUsecase domainport.UserUsecase
}

// NewController creates a new instance of the Controller struct with the provided settings.
func NewController(
logger *slog.Logger,
service *security.Service,
userUsecase domainport.UserUsecase,
) *Controller {
return &Controller{
logger:      logger,
service:     service,
userUsecase: userUsecase,
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

result, err := c.service.BasicAuth(username, password)
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

c.ensureUser(ctx.Request.Context(), result.Email, model.IdentityProviderBasic)

ctx.JSON(http.StatusOK, v1auth.AuthnTokenResponse{
Token: result.Token,
})
}

// ensureUser creates or updates a user record on login.
// Failures are logged but do not block the login flow.
func (c *Controller) ensureUser(ctx context.Context, email, provider string) {
existing, err := c.userUsecase.GetUserByEmail(ctx, email)
if err == nil && existing != nil {
existing.Metadata.UpdatedAt = time.Now()

if saveErr := c.userUsecase.SaveUser(ctx, existing); saveErr != nil {
c.logger.Warn("failed to update user on login",
slog.String("email", email),
slog.Any("error", saveErr),
)
}

return
}

newUser := model.NewUserWithIdentity(provider, email, email, email)

if saveErr := c.userUsecase.SaveUser(ctx, newUser); saveErr != nil {
c.logger.Warn("failed to create user on login",
slog.String("email", email),
slog.Any("error", saveErr),
)
}
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
