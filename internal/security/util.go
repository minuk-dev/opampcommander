package security

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// GetUser retrieves the user from the Gin context.
func GetUser(ctx *gin.Context) *User {
	if ctx == nil {
		return nil
	}

	user, exists := ctx.Get("user")
	if !exists {
		return nil
	}

	u, ok := user.(*User)
	if !ok {
		return nil
	}

	return u
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
		}
		authenticatedPrefix = []string{
			"/api/v1",
		}
	)

	return func(ctx *gin.Context) {
		if hasAnyPrefix(ctx.Request.URL.Path, bypassPrefix) {
			ctx.Next()

			return
		}

		user := &User{
			Authenticated: false,
			Email:         nil,
		}
		// Extract the JWT token from the request header
		tokenString := ctx.GetHeader("Authorization")
		if strings.HasPrefix(tokenString, "Bearer ") {
			tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		} else {
			// If the token does not start with "Bearer ", it is not a valid JWT token
			tokenString = ""
		}

		if tokenString != "" {
			claims, err := service.ValidateToken(tokenString)
			if err != nil {
				ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "unauthorized",
				})
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

	ctx.Set("user", user)
}

func hasAnyPrefix(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}
