//go:build e2e

package apiserver_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

const (
	e2eOTelCollectorImage = "otel/opentelemetry-collector-contrib:0.115.1"
	testAdminUsername     = "test-admin"
	testAdminPassword     = "test-password"
	testAdminEmail        = "test@test.com"
	testJWTSigningKey     = "test-secret-key"
	testJWTIssuer         = "e2e-test"
)

func setupAPIServer(t *testing.T, port int, mongoURI, dbName string) (func(), string) {
	t.Helper()

	managementPort, err := testutil.GetFreeTCPPort()
	require.NoError(t, err)

	//exhaustruct:ignore
	settings := config.ServerSettings{
		Address:  fmt.Sprintf("0.0.0.0:%d", port),
		ServerID: config.ServerID(fmt.Sprintf("test-server-%d", port)),
		EventSettings: config.EventSettings{
			ProtocolType: config.EventProtocolTypeInMemory,
		},
		DatabaseSettings: config.DatabaseSettings{
			Type:           config.DatabaseTypeMongoDB,
			Endpoints:      []string{mongoURI},
			ConnectTimeout: 10 * time.Second,
			DatabaseName:   dbName,
			DDLAuto:        true,
		},
		//exhaustruct:ignore
		AuthSettings: config.AuthSettings{
			//exhaustruct:ignore
			AdminSettings: config.AdminSettings{
				Username: testAdminUsername,
				Password: testAdminPassword,
				Email:    testAdminEmail,
			},
			//exhaustruct:ignore
			JWTSettings: config.JWTSettings{
				SigningKey:  testJWTSigningKey,
				Issuer:     testJWTIssuer,
				Expiration: 24 * time.Hour,
				Audience:   []string{"test"},
			},
		},
		//exhaustruct:ignore
		ManagementSettings: config.ManagementSettings{
			Address: fmt.Sprintf(":%d", managementPort),
			//exhaustruct:ignore
			ObservabilitySettings: config.ObservabilitySettings{
				//exhaustruct:ignore
				Log: config.LogSettings{
					Format: config.LogFormatText,
				},
			},
		},
	}

	server := apiserver.New(settings)
	serverCtx, cancel := context.WithCancel(t.Context())

	go func() {
		_ = server.Run(serverCtx)
	}()

	return cancel, fmt.Sprintf("http://localhost:%d", port)
}

func waitForAPIServerReady(t *testing.T, baseURL string) {
	t.Helper()

	c := client.New(baseURL)
	require.Eventually(t, func() bool {
		return c.Ping() == nil
	}, 15*time.Second, 500*time.Millisecond, "API server should start")
}

func getAuthToken(t *testing.T, baseURL string) string {
	t.Helper()

	c := client.New(baseURL)
	resp, err := c.AuthService.GetAuthTokenByBasicAuth(testAdminUsername, testAdminPassword)
	require.NoError(t, err)

	return resp.Token
}

func listAgents(t *testing.T, baseURL string) []v1.Agent {
	t.Helper()

	c := createOpampClient(t, baseURL)
	resp, err := c.AgentService.ListAgents(t.Context(), "default")
	require.NoError(t, err)

	return resp.Items
}

func getAgentByID(t *testing.T, baseURL string, uid uuid.UUID) *v1.Agent {
	t.Helper()

	c := createOpampClient(t, baseURL)
	agent, err := c.AgentService.GetAgent(t.Context(), "default", uid)
	require.NoError(t, err)

	return agent
}

func tryGetAgentByID(baseURL string, uid uuid.UUID) (*v1.Agent, error) {
	c := client.New(baseURL)

	resp, err := c.AuthService.GetAuthTokenByBasicAuth(testAdminUsername, testAdminPassword)
	if err != nil {
		return nil, err
	}

	c.SetAuthToken(resp.Token)

	return c.AgentService.GetAgent(context.Background(), "default", uid)
}

func findAgentByUID(agents []v1.Agent, uid uuid.UUID) *v1.Agent {
	for i := range agents {
		if agents[i].Metadata.InstanceUID == uid {
			return &agents[i]
		}
	}

	return nil
}

func createCollectorConfig(t *testing.T, cacheDir string, apiPort int, collectorUID uuid.UUID) string {
	t.Helper()

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
`, apiPort, collectorUID.String())

	configPath := filepath.Join(cacheDir, fmt.Sprintf("collector-config-%s.yaml", collectorUID.String()))
	err := os.WriteFile(configPath, []byte(configContent), 0600)
	require.NoError(t, err)

	return configPath
}

func startOTelCollector(t *testing.T, configPath string) testcontainers.Container {
	t.Helper()

	//exhaustruct:ignore
	req := testcontainers.ContainerRequest{
		Image:        e2eOTelCollectorImage,
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
	container, err := testcontainers.GenericContainer(t.Context(), testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	return container
}
