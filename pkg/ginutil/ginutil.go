// Package ginutil provides utility functions for working with Gin HTTP framework.
package ginutil

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetQueryInt64 retrieves an int64 query parameter from the Gin context.
func GetQueryInt64(c *gin.Context, key string, defaultValue int64) (int64, error) {
	valueStr := c.Query(key)
	if valueStr == "" {
		return defaultValue, nil
	}

	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to get query parameter %s: %w", key, err)
	}

	return value, nil
}

// GetErrorTypeURI constructs a standardized error type URI.
func GetErrorTypeURI(c *gin.Context) string {
	return c.Request.Host + c.FullPath()
}
