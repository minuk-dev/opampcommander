package security

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	userport "github.com/minuk-dev/opampcommander/internal/domain/user/port"
)

const namespaceScopedPrefix = "/api/v1/namespaces/"

// NewAuthorizationMiddleware creates a Gin middleware that enforces
// namespace-scoped RBAC for protected resources.
// The adminEmail user bypasses all RBAC checks (super-admin).
func NewAuthorizationMiddleware(
	rbacUsecase userport.RBACUsecase,
	userUsecase userport.UserUsecase,
	adminEmail string,
	logger *slog.Logger,
) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		fullPath := ctx.FullPath()

		if !strings.HasPrefix(fullPath, namespaceScopedPrefix) {
			ctx.Next()

			return
		}

		user, err := GetUser(ctx)
		if err != nil || !user.Authenticated || user.Email == nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
			})

			return
		}

		// Super-admin bypass.
		if *user.Email == adminEmail {
			ctx.Next()

			return
		}

		namespace := ctx.Param("namespace")
		resource, action := extractResourceAndAction(fullPath, ctx.Request.Method)

		if resource == "" || action == "" {
			// If the path has a resource segment but it's not in our mapping,
			// deny by default — prevents silent bypass when new endpoints are added
			// without registering them in getResourceSingular or the method switch.
			if hasNamespaceResourceSegment(fullPath) {
				ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})

				return
			}

			ctx.Next()

			return
		}

		enforcePermission(ctx, rbacUsecase, userUsecase, logger, *user.Email, namespace, resource, action)
	}
}

func enforcePermission(
	ctx *gin.Context,
	rbacUsecase userport.RBACUsecase,
	userUsecase userport.UserUsecase,
	logger *slog.Logger,
	email, namespace, resource, action string,
) {
	userModel, err := userUsecase.GetUserByEmail(ctx, email)
	if err != nil {
		logger.WarnContext(ctx, "authorization: user lookup failed",
			slog.String("email", email),
			slog.Any("error", err),
		)
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "forbidden",
		})

		return
	}

	allowed, err := rbacUsecase.CheckPermission(ctx,
		userModel.Metadata.UID, namespace, resource, action)
	if err != nil {
		logger.ErrorContext(ctx, "authorization: permission check failed",
			slog.String("user", userModel.Metadata.UID.String()),
			slog.String("namespace", namespace),
			slog.String("resource", resource),
			slog.String("action", action),
			slog.Any("error", err),
		)
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "forbidden",
		})

		return
	}

	if !allowed {
		logger.InfoContext(ctx, "authorization: access denied",
			slog.String("user", userModel.Metadata.UID.String()),
			slog.String("namespace", namespace),
			slog.String("resource", resource),
			slog.String("action", action),
		)
		ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "insufficient permissions",
		})

		return
	}

	ctx.Next()
}

// hasNamespaceResourceSegment reports whether fullPath contains a resource segment
// under /api/v1/namespaces/:namespace/, i.e. has at least 6 slash-separated parts.
// Paths shorter than this (e.g. /api/v1/namespaces/:namespace) have no resource
// segment and are not subject to per-resource RBAC checks.
func hasNamespaceResourceSegment(fullPath string) bool {
	const resourceSegmentMinParts = 6

	return len(strings.Split(fullPath, "/")) >= resourceSegmentMinParts
}

// getResourceSingular returns the singular RBAC resource name for a URL resource plural.
func getResourceSingular(plural string) (string, bool) {
	switch plural {
	case "agents":
		return "agent", true
	case "agentgroups":
		return "agentgroup", true
	case "agentpackages":
		return "agentpackage", true
	case "certificates":
		return "certificate", true
	case "agentremoteconfigs":
		return "agentremoteconfig", true
	case "rolebindings":
		return "rolebinding", true
	default:
		return "", false
	}
}

// extractResourceAndAction maps a Gin full path and HTTP method to an RBAC
// (resource, action) pair.
//
// Expected path format: /api/v1/namespaces/:namespace/<resourcePlural>[/...].
func extractResourceAndAction(fullPath, method string) (string, string) {
	// parts: ["", "api", "v1", "namespaces", ":namespace", "<resource>", ...]
	parts := strings.Split(fullPath, "/")

	const minParts = 6
	if len(parts) < minParts {
		return "", ""
	}

	resourcePlural := parts[5]

	resource, ok := getResourceSingular(resourcePlural)
	if !ok {
		return "", ""
	}

	isCollection := len(parts) == minParts ||
		(len(parts) == minParts+1 && parts[minParts] == "search")

	var action string

	switch method {
	case http.MethodGet:
		if isCollection {
			action = "LIST"
		} else {
			action = "GET"
		}
	case http.MethodPost:
		action = "CREATE"
	case http.MethodPut:
		action = "UPDATE"
	case http.MethodDelete:
		action = "DELETE"
	}

	return resource, action
}

// GetUserID is a helper that resolves the authenticated user's UUID.
func GetUserID(ctx *gin.Context, userUsecase userport.UserUsecase) (uuid.UUID, error) {
	user, err := GetUser(ctx)
	if err != nil {
		return uuid.Nil, err
	}

	if user.Email == nil {
		return uuid.Nil, ErrInvalidUserInContext
	}

	userModel, err := userUsecase.GetUserByEmail(ctx, *user.Email)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return userModel.Metadata.UID, nil
}
