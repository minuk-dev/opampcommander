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
		authenticatedPrefixList = []string{
			"/api/v1/agents",
			"/api/v1/connections",
			"/api/v1/commands",
		}
	)

	return func(ctx *gin.Context) {
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
			if err == nil {
				user = &User{
					Authenticated: true,
					Email:         &claims.Email,
				}
			}
		}

		if !user.Authenticated {
			// Check if the request path is in the authenticated prefix list
			for _, prefix := range authenticatedPrefixList {
				if strings.HasPrefix(ctx.Request.URL.Path, prefix) {
					// If the path requires authentication, return 401 Unauthorized
					ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
						"error": "unauthorized",
					})

					return
				}
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
