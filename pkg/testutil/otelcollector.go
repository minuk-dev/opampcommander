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
	otelCollectorImage = "otel/opentelemetry-collector-contrib:0.115.1"
)

type OTelCollector struct {
	*Base
	testcontainers.Container

	UID        uuid.UUID
	configPath string
}

func (b *Base) StartOTelCollector(opampPort int) *OTelCollector {
	b.t.Helper()

	cacheDir := b.CacheDir
	instanceUID := uuid.New()

	configContent := fmt.Sprintf(`receivers:
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

	configPath := filepath.Join(
		cacheDir, fmt.Sprintf("collector-config-%s.yaml", instanceUID.String()))
	err := os.WriteFile(configPath, []byte(configContent), 0600)
	require.NoError(b.t, err)

	//exhaustruct:ignore
	req := testcontainers.ContainerRequest{
		Image:        otelCollectorImage,
		ExposedPorts: []string{"4317/tcp", "4318/tcp"},
		Files: []testcontainers.ContainerFile{
			//exhaustruct:ignore
			{
				HostFilePath:      configPath,
				ContainerFilePath: "/etc/otel-collector-config.yaml",
				FileMode:          0644,
			},
		},
		Cmd:        []string{"--config=/etc/otel-collector-config.yaml"},
		WaitingFor: wait.ForLog("Everything is ready").WithStartupTimeout(60 * time.Second),
		ExtraHosts: []string{"host.docker.internal:host-gateway"},
	}

	//exhaustruct:ignore
	container, err := testcontainers.GenericContainer(
		b.t.Context(), testcontainers.GenericContainerRequest{
			ContainerRequest: req,
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
