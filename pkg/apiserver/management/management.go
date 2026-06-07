// Package management provides core management interfaces and types.
package management

import (
	"net/http"
)

// RouteInfo defines a single management route configuration.
type RouteInfo struct {
	Method  string
	Path    string
	Handler http.Handler
}

// RoutesInfo is a collection of management routes.
type RoutesInfo []RouteInfo

// HTTPHandler is an interface for components that provide management HTTP routes.
type HTTPHandler interface {
	RoutesInfos() RoutesInfo
}

// RoutesInfoWrapper wraps ManagementRoutesInfo to implement ManagementHTTPHandler.
type RoutesInfoWrapper struct {
	routesInfo RoutesInfo
}

// NewRoutesInfoWrapper creates a new ManagementRoutesInfoWrapper.
func NewRoutesInfoWrapper(routesInfo RoutesInfo) *RoutesInfoWrapper {
	return &RoutesInfoWrapper{
		routesInfo: routesInfo,
	}
}

// RoutesInfos returns the wrapped management routes.
func (w *RoutesInfoWrapper) RoutesInfos() RoutesInfo {
	return w.routesInfo
}
