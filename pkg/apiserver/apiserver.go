// Package apiserver is the public entry point for the opampcommander apiserver.
//
// It is a thin, FX-free facade over the composition root in
// github.com/minuk-dev/opampcommander/pkg/apiserver/internal/app. Keeping FX out
// of this package (and every other public package) is enforced by depguard.
package apiserver

import (
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/internal/app"
)

// Server represents the apiserver application and its lifecycle.
type Server = app.Server

// New creates a new apiserver Server from the given settings.
func New(settings config.ServerSettings) *Server {
	return app.New(settings)
}

// VisualizeError renders an FX dependency-graph error into a human-readable form,
// so callers can pretty-print startup failures without importing FX directly.
func VisualizeError(err error) (string, error) {
	//nolint:wrapcheck // thin pass-through to the composition root's VisualizeError
	return app.VisualizeError(err)
}
