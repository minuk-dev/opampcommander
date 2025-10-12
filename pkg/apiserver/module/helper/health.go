package helper

import (
	"context"
)

// HealthService is a service that aggregates multiple HealthIndicators to provide overall health and readiness status.
type HealthService struct {
	indicators []HealthIndicator
}

// NewHealthService creates a new HealthService instance.
func NewHealthService(
	indicators []HealthIndicator,
) *HealthService {
	return &HealthService{
		indicators: indicators,
	}
}

// IsReady checks the readiness of all registered health indicators.
func (s *HealthService) IsReady(ctx context.Context) bool {
	for _, indicator := range s.indicators {
		if !indicator.IsReady(ctx) {
			return false
		}
	}

	return true
}

// IsHealth checks the health of all registered health indicators.
func (s *HealthService) IsHealth(ctx context.Context) bool {
	for _, indicator := range s.indicators {
		if !indicator.IsHealth(ctx) {
			return false
		}
	}

	return true
}
