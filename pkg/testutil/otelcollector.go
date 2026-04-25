package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	otelCollectorImage          = "otel/opentelemetry-collector-contrib:0.115.1"
	otelCollectorStartupTimeout = 60 * time.Second
	collectorConfigFileMode     = 0o600
	collectorContainerFileMode  = 0o644
)

// OTelCollector wraps a testcontainer running an OpenTelemetry Collector.
type OTelCollector struct {
	*Base
	testcontainers.Container

	UID        uuid.UUID
	configPath string
}

func collectorConfigContent(opampPort int, instanceUID uuid.UUID) string {
	return fmt.Sprintf(`receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:

exporters:
  debug:
    verbosity: basic

extensions:
  opamp:
    server:
      ws:
        endpoint: ws://host.docker.internal:%d/api/v1/opamp
        tls:
          insecure: true
        headers:
          X-Test-Header: e2e-test
    instance_uid: %s

service:
  extensions: [opamp]
  telemetry:
    logs:
      level: info
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [debug]
    metrics:
      receivers: [otlp]
      processors: [batch]
      exporters: [debug]
    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [debug]
`, opampPort, instanceUID.String())
}

func buildCollectorContainerRequest(configPath string) testcontainers.ContainerRequest {
	//exhaustruct:ignore
	return testcontainers.ContainerRequest{
		Image:        otelCollectorImage,
		ExposedPorts: []string{"4317/tcp", "4318/tcp"},
		Files: []testcontainers.ContainerFile{
			//exhaustruct:ignore
			{
				HostFilePath:      configPath,
				ContainerFilePath: "/etc/otel-collector-config.yaml",
				FileMode:          collectorContainerFileMode,
			},
		},
		Cmd:        []string{"--config=/etc/otel-collector-config.yaml"},
		WaitingFor: wait.ForLog("Everything is ready").WithStartupTimeout(otelCollectorStartupTimeout),
		ExtraHosts: []string{"host.docker.internal:host-gateway"},
	}
}

// StartOTelCollector starts an OTel Collector container connected to the given OpAMP port.
func (b *Base) StartOTelCollector(opampPort int) *OTelCollector {
	b.t.Helper()

	instanceUID := uuid.New()
	configPath := filepath.Join(b.CacheDir, fmt.Sprintf("collector-config-%s.yaml", instanceUID.String()))

	err := os.WriteFile(configPath, []byte(collectorConfigContent(opampPort, instanceUID)), collectorConfigFileMode)
	require.NoError(b.t, err)

	//exhaustruct:ignore
	container, err := testcontainers.GenericContainer(
		b.t.Context(), testcontainers.GenericContainerRequest{
			ContainerRequest: buildCollectorContainerRequest(configPath),
			Started:          true,
		})
	require.NoError(b.t, err)

	return &OTelCollector{
		Base:       b,
		Container:  container,
		UID:        instanceUID,
		configPath: configPath,
	}
}
