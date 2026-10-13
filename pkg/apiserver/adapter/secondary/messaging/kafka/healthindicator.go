package kafka

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/IBM/sarama"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/management/healthcheck"
)

const (
	healthCacheTTL   = 5 * time.Second
	healthTimeout    = 500 * time.Millisecond
	healthNetTimeout = 200 * time.Millisecond
)

// HealthIndicator reports broker loss without draining or restarting a server
// that can still serve REST requests and its locally connected agents.
type HealthIndicator struct {
	brokers   []string
	mu        sync.Mutex
	checkedAt time.Time
	result    healthcheck.Health
}

var _ healthcheck.HealthIndicator = (*HealthIndicator)(nil)

// NewHealthIndicator creates a cached Kafka metadata probe.
func NewHealthIndicator(brokers []string) *HealthIndicator {
	return &HealthIndicator{
		brokers:   append([]string(nil), brokers...),
		mu:        sync.Mutex{},
		checkedAt: time.Time{},
		result:    healthcheck.Health{},
	}
}

// Name identifies this dependency in the health response body.
func (*HealthIndicator) Name() string { return "Kafka" }

// Readiness remains ready because a broker outage only prevents cross-server delivery.
func (*HealthIndicator) Readiness(context.Context) healthcheck.Readiness {
	return healthcheck.Readiness{Ready: true, Reason: ""}
}

// Health caches successes and failures; concurrent polls share the same probe.
func (i *HealthIndicator) Health(ctx context.Context) healthcheck.Health {
	i.mu.Lock()
	defer i.mu.Unlock()

	if time.Since(i.checkedAt) < healthCacheTTL {
		return i.result
	}

	result := healthcheck.Health{Healthy: true, Degraded: false, Reason: ""}

	err := i.probe(ctx)
	if err != nil {
		result.Degraded = true
		result.Reason = "cross-server event delivery unavailable: " + err.Error()
	}

	// A cancelled poll must not poison the shared dependency status.
	if ctx.Err() == nil {
		i.result = result
		i.checkedAt = time.Now()
	}

	return result
}

func (i *HealthIndicator) probe(ctx context.Context) error {
	err := ctx.Err()
	if err != nil {
		return fmt.Errorf("cancelled Kafka metadata probe: %w", err)
	}

	timeout := healthTimeout
	if deadline, ok := ctx.Deadline(); ok {
		timeout = min(timeout, time.Until(deadline))
	}

	cfg := sarama.NewConfig()
	cfg.ApiVersionsRequest = false
	cfg.Metadata.Full = false
	cfg.Metadata.RefreshFrequency = 0
	cfg.Metadata.AllowAutoTopicCreation = false
	cfg.Metadata.Retry.Max = 0
	cfg.Metadata.Timeout = timeout
	cfg.Net.DialTimeout = min(healthNetTimeout, timeout)
	cfg.Net.ReadTimeout = min(healthNetTimeout, timeout)
	cfg.Net.WriteTimeout = min(healthNetTimeout, timeout)

	client, err := sarama.NewClient(i.brokers, cfg)
	if err != nil {
		return fmt.Errorf("create Kafka metadata client: %w", err)
	}

	defer func() { _ = client.Close() }()

	err = client.RefreshMetadata()
	if err != nil {
		return fmt.Errorf("fetch Kafka metadata: %w", err)
	}

	return nil
}
