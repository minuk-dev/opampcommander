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

func collectorConfigContentHTTP(opampPort int, instanceUID uuid.UUID) string {
	return fmt.Sprintf(`receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317

processors:
  batch:

exporters:
  debug:
    verbosity: detailed

extensions:
  opamp:
    server:
      http:
        endpoint: http://host.docker.internal:%d/api/v1/opamp
        headers:
          User-Agent: "e2e-test-collector-http"
    instance_uid: %s

service:
  extensions: [opamp]
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [debug]
`, opampPort, instanceUID.String())
}

func formatResourceAttrsForConfig(attrs map[string]string) string {
	var result string
	for k, v := range attrs {
		result += fmt.Sprintf("      - key: %s\n        value: %s\n        action: upsert\n", k, v)
	}

	return result
}

func formatResourceAttrsForTelemetry(attrs map[string]string) string {
	var result string
	for k, v := range attrs {
		result += fmt.Sprintf("      %s: %s\n", k, v)
	}

	return result
}

func collectorConfigContentWithAttributes(opampPort int, instanceUID uuid.UUID, resourceAttrs map[string]string) string {
	return fmt.Sprintf(`receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:
  resource:
    attributes:
%s
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
    instance_uid: %s

service:
  extensions: [opamp]
  telemetry:
    logs:
      level: info
    resource:
%s
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch, resource]
      exporters: [debug]
    metrics:
      receivers: [otlp]
      processors: [batch, resource]
      exporters: [debug]
    logs:
      receivers: [otlp]
      processors: [batch, resource]
      exporters: [debug]
`, formatResourceAttrsForConfig(resourceAttrs), opampPort, instanceUID.String(), formatResourceAttrsForTelemetry(resourceAttrs))
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

// StartOTelCollectorHTTP starts an OTel Collector container using HTTP polling to connect to the given OpAMP port.
func (b *Base) StartOTelCollectorHTTP(opampPort int) *OTelCollector {
	b.t.Helper()

	instanceUID := uuid.New()
	configPath := filepath.Join(b.CacheDir, fmt.Sprintf("collector-config-http-%s.yaml", instanceUID.String()))

	err := os.WriteFile(configPath, []byte(collectorConfigContentHTTP(opampPort, instanceUID)), collectorConfigFileMode)
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

// StartOTelCollectorWithAttributes starts an OTel Collector container with custom resource attributes.
func (b *Base) StartOTelCollectorWithAttributes(opampPort int, resourceAttrs map[string]string) *OTelCollector {
	b.t.Helper()

	instanceUID := uuid.New()
	configPath := filepath.Join(b.CacheDir, fmt.Sprintf("collector-config-%s.yaml", instanceUID.String()))

	err := os.WriteFile(configPath, []byte(collectorConfigContentWithAttributes(opampPort, instanceUID, resourceAttrs)), collectorConfigFileMode)
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
