package security

import (
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
	return func(ctx *gin.Context) {
		user := &User{
			Authenticated: false,
			Email:         nil,
		}
		// Extract the JWT token from the request header
		tokenString := ctx.GetHeader("Authorization")
		if tokenString != "" {
			claims, err := service.ValidateToken(tokenString)
			if err == nil {
				user = &User{
					Authenticated: true,
					Email:         &claims.Email,
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
