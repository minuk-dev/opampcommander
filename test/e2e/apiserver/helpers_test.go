//go:build e2e

package apiserver_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	v1 "github.com/minuk-dev/opampcommander/api/v1"
	"github.com/minuk-dev/opampcommander/pkg/apiserver"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/minuk-dev/opampcommander/pkg/testutil"
)

const (
	testAdminUsername = "test-admin"
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

