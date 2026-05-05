// Package github provides the GitHub oauth2 authentication controller for the opampcommander.
package github

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	v1auth "github.com/minuk-dev/opampcommander/api/v1/auth"
	"github.com/minuk-dev/opampcommander/internal/domain/port"
	usermodel "github.com/minuk-dev/opampcommander/internal/domain/user/model"
	userport "github.com/minuk-dev/opampcommander/internal/domain/user/port"
	"github.com/minuk-dev/opampcommander/internal/security"
)

// Controller is a struct that implements the GitHub OAuth2 authentication controller.
type Controller struct {
	logger      *slog.Logger
	service     *security.Service
	userUsecase userport.UserUsecase
	rbacUsecase userport.RBACUsecase
}

// NewController creates a new instance of the Controller struct with the provided settings.
func NewController(
	logger *slog.Logger,
	service *security.Service,
	userUsecase userport.UserUsecase,
	rbacUsecase userport.RBACUsecase,
) *Controller {
	return &Controller{
		logger:      logger,
		service:     service,
		userUsecase: userUsecase,
		rbacUsecase: rbacUsecase,
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
			Path:        "/api/v1/auth/github/authcode",
			Handler:     "http.github.AuthCodeURL",
			HandlerFunc: c.AuthCodeURL,
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
	authcodeURL, err := c.service.AuthCodeURL("")
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
	authcodeURL, err := c.service.AuthCodeURL("")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to generate state",
			"details": fmt.Sprintf("error: %v", err),
		})

		return
	}

	ctx.JSON(http.StatusOK, v1auth.OAuth2AuthCodeURLResponse{
		URL: authcodeURL,
	})
}

// AuthCodeURL returns the GitHub OAuth2 authentication URL bound to a CLI loopback redirect.
// The provided redirect URI must point to a loopback host (127.0.0.1 / ::1 / localhost).
// On callback the server will redirect the browser to redirect_uri?token=...&refreshToken=...
// instead of returning JSON.
//
// @Summary GitHub OAuth2 Auth Code URL with CLI loopback redirect
// @Tags auth, github
// @Description Returns an OAuth2 authorization URL whose state encodes a CLI loopback redirect URI.
// @Accept json
// @Produce json
// @Param redirect_uri query string true "Loopback redirect URI (http(s)://127.0.0.1:PORT/...)"
// @Success 200 {object} OAuth2AuthCodeURLResponse
// @Failure 400 {object} map[string]any
// @Failure 500 {object} map[string]any
// @Router /api/v1/auth/github/authcode [get].
func (c *Controller) AuthCodeURL(ctx *gin.Context) {
	redirectURI := ctx.Query("redirect_uri")
	if redirectURI == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "redirect_uri is required",
		})

		return
	}

	err := validateLoopbackRedirect(redirectURI)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid redirect_uri",
			"details": err.Error(),
		})

		return
	}

	authcodeURL, err := c.service.AuthCodeURL(redirectURI)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to generate state",
			"details": fmt.Sprintf("error: %v", err),
		})

		return
	}

	ctx.JSON(http.StatusOK, v1auth.OAuth2AuthCodeURLResponse{
		URL: authcodeURL,
	})
}

// errInvalidScheme indicates the redirect_uri scheme is not http/https.
var errInvalidScheme = errors.New("redirect_uri scheme must be http or https")

// errNonLoopbackHost indicates the redirect_uri host is not a loopback address.
var errNonLoopbackHost = errors.New("redirect_uri host must be a loopback address (127.0.0.1, ::1, localhost)")

// validateLoopbackRedirect ensures the redirect URI points to a loopback host.
// Only loopback hosts are accepted to avoid serving as an open redirect for token leakage.
func validateLoopbackRedirect(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%w: got %q", errInvalidScheme, parsed.Scheme)
	}

	host := parsed.Hostname()
	switch host {
	case "127.0.0.1", "::1", "localhost":
		return nil
	default:
		return fmt.Errorf("%w: got %q", errNonLoopbackHost, host)
	}
}

// Callback handles the callback from GitHub after the user has authenticated.
// If the state encoded a CLI loopback redirect, the browser is redirected there with the
// tokens as query parameters. Otherwise the tokens are returned as JSON.
//
// @Summary	GitHub OAuth2 Callback
// @Tags auth, github
// @Description Exchanges the code received from GitHub for an authentication token.
// @Accept json
// @Produce json
// @Param state query string true "State parameter to prevent CSRF attacks"
// @Param code query string true "Code received from GitHub after authentication"
// @Success 200 {object} AuthnTokenResponse
// @Success 302
// @Failure 500 {object} map[string]any
// @Router /auth/github/callback [get].
func (c *Controller) Callback(ctx *gin.Context) {
	state := ctx.Query("state")
	code := ctx.Query("code")

	cliRedirect, _ := c.service.CLIRedirectFromState(state)

	result, err := c.service.Exchange(ctx.Request.Context(), state, code)
	if err != nil {
		c.handleCallbackError(ctx, cliRedirect, "failed to exchange code for token", err)

		return
	}

	c.ensureUser(ctx.Request.Context(), result.Email, usermodel.IdentityProviderGitHub, result.Groups)

	if cliRedirect != "" {
		c.redirectToLoopback(ctx, cliRedirect, result)

		return
	}

	ctx.JSON(http.StatusOK, v1auth.AuthnTokenResponse{
		Token:        result.Token,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    v1.NewTime(result.ExpiresAt),
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
		Expiry:                  v1.NewTime(dar.Expiry),
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

	result, err := c.service.ExchangeDeviceAuth(
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

	c.ensureUser(ctx.Request.Context(), result.Email, usermodel.IdentityProviderGitHub, result.Groups)

	ctx.JSON(http.StatusOK, v1auth.AuthnTokenResponse{
		Token:        result.Token,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    v1.NewTime(result.ExpiresAt),
	})
}

// redirectToLoopback redirects the browser to the CLI loopback URI with the tokens as query
// parameters. Loopback delivery is the standard pattern for CLI OAuth (see RFC 8252).
func (c *Controller) redirectToLoopback(ctx *gin.Context, cliRedirect string, result security.LoginResult) {
	target, err := url.Parse(cliRedirect)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "invalid cliRedirect URI",
			"details": err.Error(),
		})

		return
	}

	params := target.Query()
	params.Set("token", result.Token)

	if result.RefreshToken != "" {
		params.Set("refreshToken", result.RefreshToken)
	}

	if !result.ExpiresAt.IsZero() {
		params.Set("expiresAt", result.ExpiresAt.UTC().Format(time.RFC3339))
	}

	target.RawQuery = params.Encode()

	ctx.Redirect(http.StatusTemporaryRedirect, target.String())
}

// handleCallbackError reports callback failures back to the CLI loopback (when present)
// or as a JSON 500 otherwise. Errors are surfaced as ?error=&error_description= params.
func (c *Controller) handleCallbackError(ctx *gin.Context, cliRedirect, message string, err error) {
	if cliRedirect == "" {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   message,
			"details": fmt.Sprintf("error: %v", err),
		})

		return
	}

	target, parseErr := url.Parse(cliRedirect)
	if parseErr != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   message,
			"details": fmt.Sprintf("error: %v", err),
		})

		return
	}

	params := target.Query()
	params.Set("error", message)
	params.Set("error_description", err.Error())
	target.RawQuery = params.Encode()

	ctx.Redirect(http.StatusTemporaryRedirect, target.String())
}

// ensureUser creates or updates a user record on login.
// Always syncs provider labels (login-type, github-org-*) and re-applies RBAC policies
// so the freshly-saved user picks up the built-in default role (and any matching bindings).
// Failures are logged but do not block the login flow.
func (c *Controller) ensureUser(ctx context.Context, email, provider string, groups []string) {
	existing, err := c.userUsecase.GetUserByEmail(ctx, email)

	switch {
	case err == nil && existing != nil:
		c.syncLabels(ctx, existing, provider, groups)

		existing.Metadata.UpdatedAt = time.Now()

		saveErr := c.userUsecase.SaveUser(ctx, existing)
		if saveErr != nil {
			c.logger.Warn("failed to update user on login",
				slog.String("email", email),
				slog.Any("error", saveErr),
			)

			return
		}

		c.syncRBACPolicies(ctx, email)

		return
	case err != nil && !errors.Is(err, port.ErrResourceNotExist):
		c.logger.Warn("failed to check user existence on login",
			slog.String("email", email),
			slog.Any("error", err),
		)

		return
	}

	newUser := usermodel.NewUserWithIdentity(provider, email, email, email)
	c.syncLabels(ctx, newUser, provider, groups)

	saveErr := c.userUsecase.SaveUser(ctx, newUser)
	if saveErr != nil {
		c.logger.Warn("failed to create user on login",
			slog.String("email", email),
			slog.Any("error", saveErr),
		)

		return
	}

	c.syncRBACPolicies(ctx, email)
}

// syncRBACPolicies re-runs the Casbin policy sync so newly persisted users/bindings take effect.
// Best-effort: failures are logged but do not block the login flow.
func (c *Controller) syncRBACPolicies(ctx context.Context, email string) {
	err := c.rbacUsecase.SyncPolicies(ctx)
	if err != nil {
		c.logger.Warn("failed to sync RBAC policies after login",
			slog.String("email", email),
			slog.Any("error", err),
		)
	}
}

// syncLabels updates the user's metadata labels to reflect the current login session.
// Existing non-provider labels are preserved.
func (c *Controller) syncLabels(ctx context.Context, user *usermodel.User, provider string, groups []string) {
	_ = ctx // reserved for future use

	// Remove stale provider-specific labels before re-setting
	for key := range user.Metadata.Labels {
		if len(key) > len(usermodel.LabelGitHubOrg) && key[:len(usermodel.LabelGitHubOrg)] == usermodel.LabelGitHubOrg {
			delete(user.Metadata.Labels, key)
		}
	}

	user.SetLabel(usermodel.LabelLoginType, provider)

	if provider == usermodel.IdentityProviderGitHub {
		for _, org := range groups {
			user.SetLabel(usermodel.LabelGitHubOrg+org, "true")
		}
	}
}
