// Package ginutil provides utilities for Gin web framework.
package ginutil

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Validation error types
var (
	ErrValidationFailed = errors.New("validation failed")
	ErrRequiredParam    = errors.New("required parameter is missing")
	ErrInvalidFormat    = errors.New("invalid parameter format")
	ErrInvalidValue     = errors.New("invalid parameter value")
)

// ParseUUID parses UUID from context parameter and validates it.
// Returns error if validation fails - caller must handle error response.
func ParseUUID(c *gin.Context, paramName string) (uuid.UUID, error) {
	value := c.Param(paramName)
	if value == "" {
		return uuid.Nil, ErrRequiredParam
	}

	parsedUUID, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, ErrInvalidFormat
	}

	return parsedUUID, nil
}

// ParseInt64 parses int64 from query parameter and validates it.
// Returns error if validation fails - caller must handle error response.
func ParseInt64(c *gin.Context, paramName string, defaultValue int64) (int64, error) {
	value := c.Query(paramName)
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, ErrInvalidFormat
	}

	if parsed < 0 {
		return 0, ErrInvalidValue
	}

	return parsed, nil
}

// ParseString parses string from parameter and validates it's not empty.
// Returns error if validation fails - caller must handle error response.
func ParseString(c *gin.Context, paramName string, required bool) (string, error) {
	value := c.Param(paramName)
	if required && value == "" {
		return "", ErrRequiredParam
	}

	return value, nil
}

// BindJSON binds JSON request body and validates it.
// Returns error if validation fails - caller must handle error response.
func BindJSON(c *gin.Context, obj any) error {
	if err := c.ShouldBindJSON(obj); err != nil {
		return ErrValidationFailed
	}
	return nil
}

// HandleValidationError handles validation errors and sends appropriate RFC 9457 response.
func HandleValidationError(c *gin.Context, paramName, paramValue string, err error, isPathParam bool) {
	switch {
	case errors.Is(err, ErrRequiredParam):
		if isPathParam {
			InvalidPathParamError(c, paramName, "", "parameter is required")
		} else {
			InvalidQueryParamError(c, paramName, "", "parameter is required")
		}
	case errors.Is(err, ErrInvalidFormat):
		if isPathParam {
			InvalidPathParamError(c, paramName, paramValue, "invalid format")
		} else {
			InvalidQueryParamError(c, paramName, paramValue, "invalid format")
		}
	case errors.Is(err, ErrInvalidValue):
		if isPathParam {
			InvalidPathParamError(c, paramName, paramValue, "invalid value")
		} else {
			InvalidQueryParamError(c, paramName, paramValue, "invalid value")
		}
	case errors.Is(err, ErrValidationFailed):
		InvalidRequestBodyError(c, err)
	default:
		InternalServerError(c, err, "An error occurred during validation.")
	}
}