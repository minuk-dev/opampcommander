// Package ginutil provides utilities for Gin web framework.
package ginutil

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/minuk-dev/opampcommander/api"
	domainport "github.com/minuk-dev/opampcommander/internal/domain/port"
)

// ErrorType represents common error types.
type ErrorType int

const (
	// ErrorTypeInvalidQuery for query parameter validation errors.
	ErrorTypeInvalidQuery ErrorType = iota
	// ErrorTypeInvalidPath for path parameter validation errors.
	ErrorTypeInvalidPath
	// ErrorTypeInvalidRequestBody for request body validation errors.
	ErrorTypeInvalidRequestBody
	// ErrorTypeResourceNotFound for resource not found errors.
	ErrorTypeResourceNotFound
	// ErrorTypeInternalServer for internal server errors.
	ErrorTypeInternalServer
)

// ErrorInfo contains information about an error.
type ErrorInfo struct {
	Type     ErrorType
	Message  string
	Location string
	Value    any
	Err      error
}

// ErrorResponse creates a standardized RFC 9457 error response.
func ErrorResponse(ctx *gin.Context, errorInfo *ErrorInfo) {
	baseURL := GetErrorTypeURI(ctx)

	status, title, detail := getErrorDetails(errorInfo.Type)

	if errorInfo.Message == "" {
		errorInfo.Message = detail
	}

	errorModel := &api.ErrorModel{
		Type:     baseURL,
		Title:    title,
		Status:   status,
		Detail:   detail,
		Instance: ctx.Request.URL.String(),
		Errors: []*api.ErrorDetail{
			{
				Message:  errorInfo.Message,
				Location: errorInfo.Location,
				Value:    errorInfo.Value,
			},
		},
	}

	ctx.JSON(status, errorModel)
}

// HandleDomainError handles domain-specific errors and returns appropriate HTTP responses.
func HandleDomainError(ctx *gin.Context, err error, fallbackMessage string) {
	baseURL := GetErrorTypeURI(ctx)

	if errors.Is(err, domainport.ErrResourceNotExist) {
		ctx.JSON(http.StatusNotFound, &api.ErrorModel{
			Type:     baseURL,
			Title:    "Not Found",
			Status:   http.StatusNotFound,
			Detail:   "The requested resource does not exist.",
			Instance: ctx.Request.URL.String(),
			Errors: []*api.ErrorDetail{
				{
					Message:  "resource not found",
					Location: "server",
					Value:    nil,
				},
			},
		})
		return
	}

	// Default to internal server error
	InternalServerError(ctx, err, fallbackMessage)
}

// InvalidQueryParamError creates an error response for invalid query parameters.
func InvalidQueryParamError(ctx *gin.Context, paramName, value string, message string) {
	ErrorResponse(ctx, &ErrorInfo{
		Type:     ErrorTypeInvalidQuery,
		Message:  message,
		Location: fmt.Sprintf("query.%s", paramName),
		Value:    value,
	})
}

// InvalidPathParamError creates an error response for invalid path parameters.
func InvalidPathParamError(ctx *gin.Context, paramName, value string, message string) {
	ErrorResponse(ctx, &ErrorInfo{
		Type:     ErrorTypeInvalidPath,
		Message:  message,
		Location: fmt.Sprintf("path.%s", paramName),
		Value:    value,
	})
}

// InvalidRequestBodyError creates an error response for invalid request body.
func InvalidRequestBodyError(ctx *gin.Context, err error) {
	ErrorResponse(ctx, &ErrorInfo{
		Type:     ErrorTypeInvalidRequestBody,
		Message:  err.Error(),
		Location: "body",
		Value:    nil,
		Err:      err,
	})
}

// InternalServerError creates an error response for internal server errors.
func InternalServerError(ctx *gin.Context, err error, detail string) {
	baseURL := GetErrorTypeURI(ctx)

	ctx.JSON(http.StatusInternalServerError, &api.ErrorModel{
		Type:     baseURL,
		Title:    "Internal Server Error",
		Status:   http.StatusInternalServerError,
		Detail:   detail,
		Instance: ctx.Request.URL.String(),
		Errors: []*api.ErrorDetail{
			{
				Message:  err.Error(),
				Location: "server",
				Value:    nil,
			},
		},
	})
}

// ResourceNotFoundError creates a standardized 404 error response.
func ResourceNotFoundError(ctx *gin.Context, resourceType, identifier string) {
	baseURL := GetErrorTypeURI(ctx)

	ctx.JSON(http.StatusNotFound, &api.ErrorModel{
		Type:     baseURL,
		Title:    "Not Found",
		Status:   http.StatusNotFound,
		Detail:   fmt.Sprintf("The requested %s does not exist.", resourceType),
		Instance: ctx.Request.URL.String(),
		Errors: []*api.ErrorDetail{
			{
				Message:  fmt.Sprintf("%s not found", resourceType),
				Location: determineLocationFromURL(ctx, identifier),
				Value:    identifier,
			},
		},
	})
}

// determineLocationFromURL determines if the identifier is from path or query.
func determineLocationFromURL(ctx *gin.Context, identifier string) string {
	// Check if identifier is in path parameters
	for _, param := range ctx.Params {
		if param.Value == identifier {
			return fmt.Sprintf("path.%s", param.Key)
		}
	}
	
	// Check if identifier is in query parameters
	for key, values := range ctx.Request.URL.Query() {
		for _, value := range values {
			if value == identifier {
				return fmt.Sprintf("query.%s", key)
			}
		}
	}
	
	return "unknown"
}

// getErrorDetails returns status code, title, and detail for each error type.
func getErrorDetails(errorType ErrorType) (int, string, string) {
	switch errorType {
	case ErrorTypeInvalidQuery:
		return http.StatusBadRequest, "Invalid Query Parameter", "One or more query parameters are invalid."
	case ErrorTypeInvalidPath:
		return http.StatusBadRequest, "Invalid Path Parameter", "One or more path parameters are invalid."
	case ErrorTypeInvalidRequestBody:
		return http.StatusBadRequest, "Invalid Request Body", "The request body is not valid JSON or does not conform to the expected schema."
	case ErrorTypeResourceNotFound:
		return http.StatusNotFound, "Not Found", "The requested resource does not exist."
	case ErrorTypeInternalServer:
		return http.StatusInternalServerError, "Internal Server Error", "An unexpected error occurred while processing the request."
	default:
		return http.StatusInternalServerError, "Unknown Error", "An unknown error occurred."
	}
}