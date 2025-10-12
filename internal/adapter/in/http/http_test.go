package http_test

import (
	"github.com/minuk-dev/opampcommander/internal/adapter/in/http"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/module/helper"
)

var (
	// Ensure HealthService implements http.HealthService interface.
	_ http.HealthService = (*helper.HealthService)(nil)

	// Ensure HealthController implements helper.Controller interface.
	_ helper.Controller = (*http.HealthController)(nil)
)
