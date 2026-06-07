package config

import (
	"github.com/minuk-dev/opampcommander/pkg/apiserver/management/observability"
)

// ManagementSettings holds the settings for the management server.
type ManagementSettings struct {
	// Address is the address to bind the management server to.
	// e.g. ":9090" or "localhost:9090"
	Address string

	// Observability holds the settings for observability features. The concrete
	// type is owned by the observability package; config only aggregates it.
	Observability observability.Config
}
