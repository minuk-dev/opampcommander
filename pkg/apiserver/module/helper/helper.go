// Package helper provides utility functions and types to assist in building the application.
package helper

import (
	"github.com/gin-gonic/gin"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/module/helper/lifecycle"
)

// Controller is an interface that defines the methods for handling HTTP requests.
type Controller interface {
	RoutesInfo() gin.RoutesInfo
}

// Runner is an alias for lifecycle.Runner for backward compatibility.
type Runner = lifecycle.Runner
