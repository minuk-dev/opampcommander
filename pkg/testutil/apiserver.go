package testutil

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"github.com/minuk-dev/opampcommander/pkg/apiserver"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	usermodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/management/observability"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/security"
	"github.com/minuk-dev/opampcommander/pkg/client"
)

const (
	apiServerStartTimeout = 60 * time.Second
	apiServerPollInterval = 500 * time.Millisecond
	dbConnectTimeout      = 10 * time.Second
	jwtExpiration         = 24 * time.Hour
	kafkaEventTopic       = "e2e.opampcommander.events"

	// DefaultAdminUsername is the admin username used in test API servers.
	DefaultAdminUsername = "test-admin"
	// DefaultAdminPassword is the admin password used in test API servers.
	DefaultAdminPassword = "test-password"
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

	Settings   config.ServerSettings
	stopServer func() // cancels the server's running context
}

// AdminUsername returns the admin username configured for this test server.
func (a *APIServer) AdminUsername() string {
	return a.Settings.Security.AdminSettings.Username
}

// AdminPassword returns the admin password configured for this test server.
func (a *APIServer) AdminPassword() string {
	return a.Settings.Security.AdminSettings.Password
}

// WaitForReady blocks until the API server responds to ping or the timeout expires.
func (a *APIServer) WaitForReady() {
	a.t.Helper()

	pingClient := client.New(a.Endpoint)

	require.Eventually(a.t, func() bool {
		err := pingClient.Ping()

		return err == nil
	}, apiServerStartTimeout, apiServerPollInterval, "API server should start")
}

// Stop signals the API server to shut down by cancelling its running context.
// The server's Run goroutine handles graceful shutdown with a built-in timeout.
func (a *APIServer) Stop() {
	a.stopServer()
}

// Client returns a pre-authenticated client pointing at this test server.
func (a *APIServer) Client() *client.Client {
	return client.New(a.Endpoint,
		client.WithBasicAuth(a.AdminUsername(), a.AdminPassword()),
	)
}

// IssueTokenForEmail signs an access JWT for the given email using the same key,
// issuer, and audience as the running test server. The token mirrors what the
// production basic/GitHub auth flows would issue, so RBAC enforcement can be
// exercised in tests for any user that already exists in the user store.
//
// The caller is responsible for ensuring a user record with this email exists —
// the authorization middleware rejects requests whose email does not resolve to a
// stored user. Use the admin client's UserService.CreateUser to seed it first.
func (a *APIServer) IssueTokenForEmail(email string) string {
	a.t.Helper()

	jwtSettings := a.Settings.Security.JWTSettings
	now := time.Now()

	claims := jwt.MapClaims{
		"email":     email,
		"tokenType": "access",
		"iss":       jwtSettings.Issuer,
		"sub":       "opampcommander",
		"aud":       jwtSettings.Audience,
		"nbf":       now.Unix(),
		"iat":       now.Unix(),
		"exp":       now.Add(jwtSettings.Expiration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString([]byte(jwtSettings.SigningKey))
	require.NoError(a.t, err, "failed to sign test JWT")

	return signed
}

// ClientAs returns a client authenticated as the given email. The user must
// already exist (see IssueTokenForEmail).
func (a *APIServer) ClientAs(email string) *client.Client {
	return client.New(a.Endpoint, client.WithBearerToken(a.IssueTokenForEmail(email)))
}

func buildServerSettings(
	serverID string, serverPort, managementPort int, mongoURI, databaseName string,
) config.ServerSettings {
	return config.ServerSettings{
		Address:  fmt.Sprintf("0.0.0.0:%d", serverPort),
		ServerID: agentmodel.ServerID(serverID),
		MetricsBackend: config.MetricsBackendSettings{
			Type:          config.MetricsBackendTypeNone,
			Address:       "",
			DefaultWindow: 0,
		},
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
		Security: security.Config{
			//exhaustruct:ignore
			AdminSettings: security.AdminSettings{
				Username: DefaultAdminUsername,
				Password: DefaultAdminPassword,
				Email:    "test@test.com",
			},
			//exhaustruct:ignore
			JWTSettings: security.JWTSettings{
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
			Observability: observability.Config{
				//exhaustruct:ignore
				Log: observability.LogSettings{
					Format: observability.LogFormatText,
				},
			},
		},
		CacheSettings: config.DefaultCacheSettings(),
		// Seed from the repository's default manifest directory so tests exercise the
		// same built-in resources a stock deployment ships.
		BootstrapSettings: config.BootstrapSettings{
			Dir:              repoInitialDir(),
			DefaultNamespace: agentmodel.DefaultNamespaceName,
			DefaultRole:      usermodel.RoleDefault,
		},
		RBACModelPath: "",
	}
}

// repoInitialDir resolves the repository's default manifest directory relative to
// this source file, so tests seed the built-in resources regardless of the test
// working directory.
func repoInitialDir() string {
	_, thisFile, _, _ := runtime.Caller(0) //nolint:dogsled
	// pkg/testutil/apiserver.go -> repo root is two levels up.
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "configs", "apiserver", "initial")
}

// StartAPIServer starts a new API server backed by the given MongoDB instance.
func (b *Base) StartAPIServer(
	mongoURI string,
	databaseName string,
) *APIServer {
	b.t.Helper()

	serverID := b.nextServerID()
	serverPort := b.GetFreeTCPPort()
	managementPort := b.GetFreeTCPPort()

	settings := buildServerSettings(serverID, serverPort, managementPort, mongoURI, databaseName)

	return b.launchAPIServer(settings, serverID, serverPort, managementPort, mongoURI)
}

// StartAPIServerWithKafka starts a new API server backed by MongoDB and Kafka for event processing.
func (b *Base) StartAPIServerWithKafka(mongoURI, kafkaBroker, databaseName string) *APIServer {
	b.t.Helper()

	serverID := b.nextServerID()
	serverPort := b.GetFreeTCPPort()
	managementPort := b.GetFreeTCPPort()

	settings := buildServerSettings(serverID, serverPort, managementPort, mongoURI, databaseName)
	settings.EventSettings = config.EventSettings{
		ProtocolType: config.EventProtocolTypeKafka,
		KafkaSettings: config.KafkaSettings{
			Brokers: []string{kafkaBroker},
			Topic:   kafkaEventTopic,
		},
	}

	return b.launchAPIServer(settings, serverID, serverPort, managementPort, mongoURI)
}

// StartStandaloneAPIServer starts a new API server in standalone mode: the
// in-memory persistence store (no MongoDB) and the in-memory event hub (no
// Kafka). It needs no external dependencies, so it is suitable for fast
// integration tests that exercise the full HTTP -> application -> domain ->
// persistence stack without Docker.
func (b *Base) StartStandaloneAPIServer() *APIServer {
	b.t.Helper()

	serverID := b.nextServerID()
	serverPort := b.GetFreeTCPPort()
	managementPort := b.GetFreeTCPPort()

	settings := buildServerSettings(serverID, serverPort, managementPort, "", "")
	settings.DatabaseSettings = config.DatabaseSettings{
		Type:           config.DatabaseTypeInMemory,
		Endpoints:      nil,
		ConnectTimeout: 0,
		DatabaseName:   "",
		DDLAuto:        false,
	}

	return b.launchAPIServer(settings, serverID, serverPort, managementPort, "")
}

// launchAPIServer constructs and starts an API server from the given settings,
// returning a handle wired with the test ports and shutdown hook.
func (b *Base) launchAPIServer(
	settings config.ServerSettings,
	serverID string,
	serverPort, managementPort int,
	mongoURI string,
) *APIServer {
	b.t.Helper()

	server := apiserver.New(settings)

	serverCtx, serverCancel := context.WithCancel(b.t.Context())

	go func() {
		_ = server.Run(serverCtx)
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
		stopServer:         serverCancel,
	}
}
