package healthcheck

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/minuk-dev/opampcommander/internal/management"
)

// HealthIndicator is an interface that defines the methods for checking the health and readiness of the service.
type HealthIndicator interface {
	Name() string
	Readiness(ctx context.Context) Readiness
	Health(ctx context.Context) Health
}

// Readiness represents the readiness status of a service.
type Readiness struct {
	// Ready indicates if the service is ready to serve requests.
	Ready bool
	// Reason provides additional information if the service is not ready.
	Reason string
}

// Health represents the health status of a service.
type Health struct {
	// Healthy indicates if the service is healthy.
	Healthy bool
	// Reason provides additional information if the service is not healthy.
	Reason string
}

// HealthHelper is a service that aggregates multiple HealthIndicators to provide overall health and readiness status.
type HealthHelper struct {
	indicators []HealthIndicator
}

var (
	_ management.ManagementHTTPHandler = (*HealthHelper)(nil)
)

// NewHealthHelper creates a new HealthService instance.
func NewHealthHelper(
	indicators []HealthIndicator,
) *HealthHelper {
	return &HealthHelper{
		indicators: indicators,
	}
}

// Readiness checks the readiness of all registered health indicators.
func (s *HealthHelper) Readiness(ctx context.Context) (bool, map[string]string) {
	reasons := make(map[string]string)
	ready := true

	for _, indicator := range s.indicators {
		indicatorReady := indicator.Readiness(ctx)
		if !indicatorReady.Ready {
			ready = false
			reasons[indicator.Name()] = indicatorReady.Reason
		}
	}

	return ready, reasons
}

// Health checks the health of all registered health indicators.
func (s *HealthHelper) Health(ctx context.Context) (bool, map[string]string) {
	reasons := make(map[string]string)
	healthy := true

	for _, indicator := range s.indicators {
		indicatorHealth := indicator.Health(ctx)
		if !indicatorHealth.Healthy {
			healthy = false
			reasons[indicator.Name()] = indicatorHealth.Reason
		}
	}

	return healthy, reasons
}

// RoutesInfo implements helper.Controller.
func (c *HealthHelper) RoutesInfos() management.ManagementRoutesInfo {
	return management.ManagementRoutesInfo{
		{
			Method:  http.MethodGet,
			Path:    "/healthz",
			Handler: http.HandlerFunc(c.IsHealth),
		},
		{
			Method:  http.MethodGet,
			Path:    "/readyz",
			Handler: http.HandlerFunc(c.IsReady),
		},
	}
}

// IsReady checks if the service is ready.
//
// @Summary Readiness Check
// @Description Check if the service is ready
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {string} string "OK"
// @Failure 503 {string} string "Service Unavailable"
// @Router /readyz [get].
func (c *HealthHelper) IsReady(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	ready, reasons := c.Readiness(ctx)
	if ready {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	err := json.NewEncoder(w).Encode(reasons)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// IsHealth checks if the service is healthy.
//
// @Summary Liveness Check
// @Description Check if the service is healthy
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {string} string "OK"
// @Failure 503 {string} string "Service Unavailable"
// @Router /healthz [get].
func (c *HealthHelper) IsHealth(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	healthy, reasons := c.Health(ctx)
	if healthy {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	err := json.NewEncoder(w).Encode(reasons)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
