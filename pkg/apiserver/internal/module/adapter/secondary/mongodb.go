package secondary

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/v2/mongo/otelmongo"
	traceapi "go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/adapter/out/persistence/mongodb"
	applicationport "github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	userport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/user/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/internal/module/helper"
)

// NewMongoDB provides the MongoDB client/database and every persistence adapter
// (repositories, transaction runner, health indicator) built on top of it.
func NewMongoDB() fx.Option {
	return fx.Options(
		fx.Provide(
			NewMongoDBClient,
			NewMongoDatabase,
			helper.AsHealthIndicator(NewMongoDBHealthIndicator),
			fx.Annotate(
				mongodb.NewTransactionRunner,
				fx.As(new(applicationport.TransactionRunner)),
			),
			fx.Annotate(mongodb.NewAgentRepository, fx.As(new(agentport.AgentPersistencePort))),
			fx.Annotate(mongodb.NewAgentGroupRepository, fx.As(new(agentport.AgentGroupPersistencePort))),
			fx.Annotate(mongodb.NewServerAdapter, fx.As(new(agentport.ServerPersistencePort))),
			fx.Annotate(mongodb.NewAgentPackageRepository, fx.As(new(agentport.AgentPackagePersistencePort))),
			fx.Annotate(mongodb.NewNamespaceRepository, fx.As(new(agentport.NamespacePersistencePort))),
			fx.Annotate(mongodb.NewAgentRemoteConfigRepository, fx.As(new(agentport.AgentRemoteConfigPersistencePort))),
			fx.Annotate(mongodb.NewCertificateRepository, fx.As(new(agentport.CertificatePersistencePort))),
			// RBAC MongoDB adapters
			fx.Annotate(mongodb.NewUserRepository, fx.As(new(userport.UserPersistencePort))),
			fx.Annotate(mongodb.NewRoleRepository, fx.As(new(userport.RolePersistencePort))),
			fx.Annotate(mongodb.NewPermissionRepository, fx.As(new(userport.PermissionPersistencePort))),
			fx.Annotate(mongodb.NewUserRoleRepository, fx.As(new(userport.UserRolePersistencePort))),
			fx.Annotate(mongodb.NewRoleBindingRepository, fx.As(new(userport.RoleBindingPersistencePort))),
		),
	)
}

// NewMongoDBClient creates a new MongoDB client with OpenTelemetry instrumentation.
func NewMongoDBClient(
	settings *config.ServerSettings,
	// meterProvider metricapi.MeterProvider,
	traceProvider traceapi.TracerProvider,
	lifecycle fx.Lifecycle,
) (*mongo.Client, error) {
	var uri string
	if len(settings.DatabaseSettings.Endpoints) > 0 {
		uri = settings.DatabaseSettings.Endpoints[0]
	} else {
		uri = "mongodb://localhost:27017"
	}

	monitor := getObservabilityForMongo(traceProvider)
	// Use OpenTelemetry MongoDB instrumentation
	clientOptions := options.Client().
		ApplyURI(uri).
		SetMonitor(monitor)

	mongoClient, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, fmt.Errorf("mongo client init failed: %w", err)
	}

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			var cancel context.CancelFunc
			if settings.DatabaseSettings.ConnectTimeout > 0 {
				ctx, cancel = context.WithTimeout(ctx, settings.DatabaseSettings.ConnectTimeout)
				defer cancel()
			}

			err := mongoClient.Ping(ctx, nil)
			if err != nil {
				return fmt.Errorf("mongo client ping failed: %w", err)
			}

			return nil
		},
		OnStop: func(ctx context.Context) error {
			err := mongoClient.Disconnect(ctx)
			if err != nil {
				return fmt.Errorf("failed to disconnect mongo client: %w", err)
			}

			return nil
		},
	})

	return mongoClient, nil
}

// NewMongoDatabase creates a new MongoDB database from the client.
func NewMongoDatabase(
	client *mongo.Client,
	settings *config.ServerSettings,
	lifecycle fx.Lifecycle,
) (*mongo.Database, error) {
	databaseName := settings.DatabaseSettings.DatabaseName
	// Use a default database name if not specified
	// You can extract database name from connection string or use a default
	if settings.DatabaseSettings.DatabaseName == "" {
		databaseName = "opampcommander"
	}

	database := client.Database(databaseName)

	if settings.DatabaseSettings.DDLAuto {
		// Register schema initialization in lifecycle
		lifecycle.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				err := mongodb.EnsureSchema(ctx, database)
				if err != nil {
					return fmt.Errorf("failed to ensure mongo schema: %w", err)
				}

				return nil
			},
			OnStop: nil,
		})
	}

	return database, nil
}

func getObservabilityForMongo(
	// meterProvider metricapi.MeterProvider,
	traceProvider traceapi.TracerProvider,
) *event.CommandMonitor {
	monitor := otelmongo.NewMonitor(
		//nolint:godox // external issue
		// TODO: Enable when https://github.com/open-telemetry/opentelemetry-go-contrib/pull/7983 merged
		// otelmongo.WithMeterProvider(meterProvider),
		otelmongo.WithTracerProvider(traceProvider),
	)

	return monitor
}
