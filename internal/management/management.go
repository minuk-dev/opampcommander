package management

import (
	"net/http"
)

type ManagementRouteInfo struct {
	Method  string
	Path    string
	Handler http.Handler
}

type ManagementRoutesInfo []ManagementRouteInfo

type ManagementHTTPHandler interface {
	RoutesInfos() ManagementRoutesInfo
}

type ManagementRoutesInfoWrapper struct {
	routesInfo ManagementRoutesInfo
}

func NewManagementRoutesInfoWrapper(routesInfo ManagementRoutesInfo) *ManagementRoutesInfoWrapper {
	return &ManagementRoutesInfoWrapper{
		routesInfo: routesInfo,
	}
}

func (w *ManagementRoutesInfoWrapper) RoutesInfos() ManagementRoutesInfo {
	return w.routesInfo
}
