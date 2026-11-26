// Package infrastructure registers startup reconciliation of built-in resources.
package infrastructure

import "go.uber.org/fx"

// New registers the manifest bootstrap hook.
func New() fx.Option {
	return fx.Module("infrastructure", fx.Invoke(registerBootstrapHook))
}
