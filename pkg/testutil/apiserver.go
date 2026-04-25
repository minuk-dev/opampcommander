package testutil

import (
	"fmt"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	"github.com/minuk-dev/opampcommander/pkg/client"
)

const (
	apiServerStartTimeout = 15 * time.Second
	apiServerPollInterval = 500 * time.Millisecond
	dbConnectTimeout      = 10 * time.Second
	jwtExpiration         = 24 * time.Hour
)

// APIServer holds a running test API server and its configuration.
type APIServer struct {
	*Base

	Server   *apiserver.Server
	ServerID string

	Endpoint string
	Port     int

	ManagementEndpoint string
	ManagementPort     int

	MongoURI string

	Settings config.ServerSettings
}

// AdminUsername returns the admin username configured for this test server.
func (a *APIServer) AdminUsername() string {
	return a.Settings.AuthSettings.AdminSettings.Username
}

// AdminPassword returns the admin password configured for this test server.
func (a *APIServer) AdminPassword() string {
	return a.Settings.AuthSettings.AdminSettings.Password
}

// WaitForReady blocks until the API server responds to ping or the timeout expires.
func (a *APIServer) WaitForReady() {
	a.t.Helper()

	c := a.Client()

	require.Eventually(a.t, func() bool {
		err := c.Ping()

		return err == nil
	}, apiServerStartTimeout, apiServerPollInterval, "API server should start")
}

// Stop shuts down the API server and fails the test on error.
func (a *APIServer) Stop() {
	a.t.Helper()
	require.NoError(a.t, a.Server.Stop(a.t.Context()))
}

// Client returns a pre-authenticated client pointing at this test server.
func (a *APIServer) Client() *client.Client {
	return client.New(a.Endpoint,
		client.WithBasicAuth(a.AdminUsername(), a.AdminPassword()),
	)
}

func buildServerSettings(
	serverID string, serverPort, managementPort int, mongoURI, databaseName string,
) config.ServerSettings {
	return config.ServerSettings{
		Address:  fmt.Sprintf("0.0.0.0:%d", serverPort),
		ServerID: config.ServerID(serverID),
		//exhaustruct:ignore
		EventSettings: config.EventSettings{
			ProtocolType: config.EventProtocolTypeInMemory,
		},
		DatabaseSettings: config.DatabaseSettings{
			Type:           config.DatabaseTypeMongoDB,
			Endpoints:      []string{mongoURI},
			ConnectTimeout: dbConnectTimeout,
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
				Expiration: jwtExpiration,
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
		CacheSettings: config.DefaultCacheSettings(),
		RBACModelPath: "",
	}
}

// StartAPIServer starts a new API server backed by the given MongoDB instance.
func (b *Base) StartAPIServer(
	mongoURI string,
	databaseName string,
) *APIServer {
	b.t.Helper()

	serverID := Identifier(b.t)
	serverPort := b.GetFreeTCPPort()
	managementPort := b.GetFreeTCPPort()

	settings := buildServerSettings(serverID, serverPort, managementPort, mongoURI, databaseName)
	server := apiserver.New(settings)

	go func() {
		_ = server.Run(b.t.Context())
	}()

	return &APIServer{
		Base:               b,
		Server:             server,
		ServerID:           serverID,
		Endpoint:           fmt.Sprintf("http://localhost:%d", serverPort),
		Port:               serverPort,
		ManagementEndpoint: fmt.Sprintf("http://localhost:%d", managementPort),
		ManagementPort:     managementPort,
		MongoURI:           mongoURI,
		Settings:           settings,
	}
}
