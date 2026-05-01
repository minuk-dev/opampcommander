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

const (
	namespaceScopedPrefix = "/api/v1/namespaces/"
	globalAPIPrefix       = "/api/v1/"
	wildcardNamespace     = "*"
)

// NewAuthorizationMiddleware creates a Gin middleware that enforces RBAC for
// both namespace-scoped (/api/v1/namespaces/:namespace/*) and global
// (/api/v1/users, /api/v1/roles, /api/v1/servers) resources.
// The adminEmail user bypasses all RBAC checks.
func NewAuthorizationMiddleware(
	rbacUsecase userport.RBACUsecase,
	userUsecase userport.UserUsecase,
	adminEmail string,
	logger *slog.Logger,
) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		fullPath := ctx.FullPath()

		if isExemptFromRBAC(fullPath) {
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

		var namespace, resource, action string

		if strings.HasPrefix(fullPath, namespaceScopedPrefix) {
			namespace = ctx.Param("namespace")
			resource, action = extractNamespacedResourceAndAction(fullPath, ctx.Request.Method)

			if resource == "" || action == "" {
				// Deny by default if a resource segment exists but isn't mapped.
				if hasNamespaceResourceSegment(fullPath) {
					ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})

					return
				}

				ctx.Next()

				return
			}
		} else {
			namespace = wildcardNamespace
			resource, action = extractGlobalResourceAndAction(fullPath, ctx.Request.Method)

			if resource == "" || action == "" {
				ctx.Next()

				return
			}
		}

		enforcePermission(ctx, rbacUsecase, userUsecase, logger, *user.Email, namespace, resource, action)
	}
}

// isExemptFromRBAC returns true for paths that skip RBAC entirely:
// authentication flows, public endpoints, self-access, namespace management,
// and RBAC management endpoints.
func isExemptFromRBAC(fullPath string) bool {
	if strings.HasPrefix(fullPath, "/auth/") ||
		strings.HasPrefix(fullPath, "/api/v1/auth/") ||
		strings.HasPrefix(fullPath, "/api/v1/rbac/") {
		return true
	}

	switch fullPath {
	case "/api/v1/ping",
		"/api/v1/version",
		"/api/v1/opamp",
		"/api/v1/users/me",
		"/api/v1/namespaces",
		"/api/v1/namespaces/:namespace":
		return true
	}

	return false
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

// hasNamespaceResourceSegment reports whether fullPath has a resource segment
// under /api/v1/namespaces/:namespace/, i.e. at least 6 slash-separated parts.
func hasNamespaceResourceSegment(fullPath string) bool {
	const resourceSegmentMinParts = 6

	return len(strings.Split(fullPath, "/")) >= resourceSegmentMinParts
}

// extractNamespacedResourceAndAction maps a namespace-scoped path and HTTP method
// to an RBAC (resource, action) pair.
// Expected format: /api/v1/namespaces/:namespace/<resourcePlural>[/...].
func extractNamespacedResourceAndAction(fullPath, method string) (string, string) {
	// parts: ["", "api", "v1", "namespaces", ":namespace", "<resource>", ...]
	parts := strings.Split(fullPath, "/")

	const minParts = 6
	if len(parts) < minParts {
		return "", ""
	}

	resource, ok := namespacedResourceSingular(parts[5])
	if !ok {
		return "", ""
	}

	isCollection := len(parts) == minParts ||
		(len(parts) == minParts+1 && parts[minParts] == "search")

	return resource, methodToAction(method, isCollection)
}

// extractGlobalResourceAndAction maps a global (non-namespaced) path and HTTP method
// to an RBAC (resource, action) pair.
// Expected format: /api/v1/<resourcePlural>[/:id].
func extractGlobalResourceAndAction(fullPath, method string) (string, string) {
	// parts: ["", "api", "v1", "<resourcePlural>", ...]
	parts := strings.Split(fullPath, "/")

	const minParts = 4
	if len(parts) < minParts {
		return "", ""
	}

	resource, ok := globalResourceSingular(parts[3])
	if !ok {
		return "", ""
	}

	isCollection := len(parts) == minParts

	return resource, methodToAction(method, isCollection)
}

func namespacedResourceSingular(plural string) (string, bool) {
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

func globalResourceSingular(plural string) (string, bool) {
	switch plural {
	case "users":
		return "user", true
	case "roles":
		return "role", true
	case "servers":
		return "server", true
	default:
		return "", false
	}
}

func methodToAction(method string, isCollection bool) string {
	switch method {
	case http.MethodGet:
		if isCollection {
			return "LIST"
		}

		return "GET"
	case http.MethodPost:
		return "CREATE"
	case http.MethodPut:
		return "UPDATE"
	case http.MethodDelete:
		return "DELETE"
	default:
		return ""
	}
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
