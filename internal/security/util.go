package security

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type userContextKeyType struct {
	key string
}

var (
	//nolint:gochecknoglobals
	userContextKey = userContextKeyType{
		key: "user",
	}
)

var (
	// ErrNilContext is returned when the context is nil.
	ErrNilContext = errors.New("nil context")
	// ErrInvalidContext is returned when the context is not a valid Gin context.
	ErrInvalidContext = errors.New("invalid context")
	// ErrInvalidUserInContext is returned when the user in the context is not valid.
	ErrInvalidUserInContext = errors.New("invalid user in context")
)

// GetUser retrieves the user from the Gin context.
//
//nolint:contextcheck
func GetUser(ctx context.Context) (*User, error) {
	if ctx == nil {
		return nil, ErrNilContext
	}

	ginContext, ok := ctx.(*gin.Context)
	if ok {
		ctx = ginContext.Request.Context()
	}

	user := ctx.Value(userContextKey)

	u, ok := user.(*User)
	if !ok {
		return nil, ErrInvalidUserInContext
	}

	return u, nil
}

// NewAnonymousUser creates a new anonymous user.
// Some operations needs an user (e.g., for audit logging) even if the user is not authenticated.
func NewAnonymousUser() *User {
	return &User{
		Authenticated: false,
		Email:         nil,
	}
}

// NewAuthJWTMiddleware creates a new Gin middleware for JWT authentication.
func NewAuthJWTMiddleware(
	service *Service,
) gin.HandlerFunc {
	var (
		bypassPrefix = []string{
			"/auth",
			"/api/v1/auth/basic",
			"/api/v1/auth/github",
			"/api/v1/ping",
			"/api/v1/opamp",
			"/api/v1/version",
			"/api/v1/agents",      // For E2E tests
			"/api/v1/agentgroups", // For E2E tests
			"/healthz",
			"/readyz",
		}
		authenticatedPrefix = []string{
			"/api/v1",
		}
	)

	return func(ctx *gin.Context) {
		if hasAnyPrefix(ctx.Request.URL.Path, bypassPrefix) {
			saveUser(ctx, NewAnonymousUser())
			ctx.Next()

			return
		}

		user := &User{
			Authenticated: false,
			Email:         nil,
		}
		// Extract the JWT token from the request header
		tokenString, found := strings.CutPrefix(ctx.GetHeader("Authorization"), "Bearer ")
		if found {
			claims, err := service.ValidateToken(tokenString)
			if err != nil {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "unauthorized",
				})

				return
			}

			user = &User{
				Authenticated: true,
				Email:         &claims.Email,
			}
		}

		if !user.Authenticated {
			if hasAnyPrefix(ctx.Request.URL.Path, authenticatedPrefix) {
				// If the path requires authentication, return 401 Unauthorized
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "unauthorized",
				})

				return
			}
		}
		// Save the user in the context
		saveUser(ctx, user)
		ctx.Next()
	}
}

// saveUser saves the user in the Gin context.
func saveUser(ctx *gin.Context, user *User) {
	if ctx == nil || user == nil {
		return
	}

	ctx.Request = ctx.Request.WithContext(
		context.WithValue(ctx.Request.Context(), userContextKey, user),
	)
}

func hasAnyPrefix(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}
