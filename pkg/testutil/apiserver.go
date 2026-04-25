package testutil

import (
	"fmt"
	"time"

	"github.com/minuk-dev/opampcommander/pkg/apiserver"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/client"
	"github.com/stretchr/testify/require"
)

const (
	apiServerStartTimeout = 15 * time.Second
)

type APIServer struct {
	*Base

	Server   apiserver.Server
	ServerID string

	Endpoint string
	Port     int

	ManagementEndpoint string
	ManagementPort     int

	MongoURI string

	Settings config.ServerSettings
}

func (a *APIServer) AdminUsername() string {
	return a.Settings.AuthSettings.AdminSettings.Username
}

func (a *APIServer) AdminPassword() string {
	return a.Settings.AuthSettings.AdminSettings.Password
}

func (a *APIServer) WaitForReady() {
	a.t.Helper()

	client := a.Client()

	require.Eventually(a.t, func() bool {
		err := client.Ping()
		return err == nil
	}, apiServerStartTimeout, 500*time.Millisecond, "API server should start")
}

func (a *APIServer) Stop() {
	a.t.Helper()
	require.NoError(a.t, a.Server.Stop(a.t.Context()))
}

func (a *APIServer) Client() *client.Client {
	return client.New(a.Endpoint,
		client.WithBasicAuth(a.AdminUsername(), a.AdminPassword()),
	)
}

func (b *Base) StartAPIServer(
	mongoURI string,
	databaseName string,
) *APIServer {
	b.t.Helper()

	serverID := Identifier(b.t)

	serverPort := b.GetFreeTCPPort()
	managementPort := b.GetFreeTCPPort()

	settings := config.ServerSettings{
		Address:  fmt.Sprintf("0.0.0.0:%d", serverPort),
		ServerID: config.ServerID(serverID),
		EventSettings: config.EventSettings{
			ProtocolType: config.EventProtocolTypeInMemory,
		},
		DatabaseSettings: config.DatabaseSettings{
			Type:           config.DatabaseTypeMongoDB,
			Endpoints:      []string{mongoURI},
			ConnectTimeout: 10 * time.Second,
			DatabaseName:   databaseName,
			DDLAuto:        true,
		},
		//exhaustruct:ignore
		AuthSettings: config.AuthSettings{
			//exhaustruct:ignore
			AdminSettings: config.AdminSettings{
				Username: "test-admin",
				Password: "test-password",
				Email:    "test@test.com",
			},
			//exhaustruct:ignore
			JWTSettings: config.JWTSettings{
				SigningKey: "test-secret-key",
				Issuer:     "e2e-test",
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
	go func() {
		_ = server.Run(b.t.Context())
	}()

	return &APIServer{
		Base:               b,
		ServerID:           serverID,
		Endpoint:           fmt.Sprintf("http://localhost:%d", serverPort),
		Port:               serverPort,
		ManagementEndpoint: fmt.Sprintf("http://localhost:%d", managementPort),
		ManagementPort:     managementPort,
		MongoURI:           mongoURI,
		Settings:           settings,
	}
}
