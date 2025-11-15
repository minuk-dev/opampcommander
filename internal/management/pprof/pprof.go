// Package pprof provides functionalities for performance profiling.
package pprof

import (
	"net/http"
	"net/http/pprof"

	"github.com/minuk-dev/opampcommander/internal/management"
)

// Handler is an HTTP handler that provides pprof endpoints.
type Handler struct{}

var (
	_ management.HTTPHandler = (*Handler)(nil)
)

// NewHandler creates a new PprofHandler instance.
func NewHandler() *Handler {
	return &Handler{}
}

// RoutesInfos implements management.HTTPHandler.
func (p *Handler) RoutesInfos() management.RoutesInfo {
	return management.RoutesInfo{
		{
			Method:  http.MethodGet,
			Path:    "/debug/pprof/",
			Handler: http.HandlerFunc(pprof.Index),
		},
		{
			Method:  http.MethodGet,
			Path:    "/debug/pprof/cmdline",
			Handler: http.HandlerFunc(pprof.Cmdline),
		},
		{
			Method:  http.MethodGet,
			Path:    "/debug/pprof/profile",
			Handler: http.HandlerFunc(pprof.Profile),
		},
		{
			Method:  http.MethodGet,
			Path:    "/debug/pprof/symbol",
			Handler: http.HandlerFunc(pprof.Symbol),
		},
		{
			Method:  http.MethodGet,
			Path:    "/debug/pprof/trace",
			Handler: http.HandlerFunc(pprof.Trace),
		},
	}
}
