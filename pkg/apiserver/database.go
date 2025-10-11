package apiserver

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/event"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/v2/mongo/otelmongo"
	metricapi "go.opentelemetry.io/otel/metric"
	traceapi "go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/internal/adapter/out/persistence/mongodb"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
)

// Controller is an interface that defines the methods for handling HTTP requests.
type Controller interface {
	RoutesInfo() gin.RoutesInfo
}

// NewMongoDBClient creates a new MongoDB client with OpenTelemetry instrumentation.
func NewMongoDBClient(
	settings *config.ServerSettings,
	meterProvider metricapi.MeterProvider,
	traceProvider traceapi.TracerProvider,
	logger *slog.Logger,
	lifecycle fx.Lifecycle,
) (*mongo.Client, error) {
	var uri string
	if len(settings.DatabaseSettings.Endpoints) > 0 {
		uri = settings.DatabaseSettings.Endpoints[0]
	} else {
		uri = "mongodb://localhost:27017"
	}

	monitor, poolMonitor, err := getObservabilityForMongo(meterProvider, traceProvider, lifecycle)
	if err != nil {
		return nil, fmt.Errorf("failed to set observability to mongo client: %w", err)
	}
	// Use OpenTelemetry MongoDB instrumentation
	clientOptions := options.Client().
		ApplyURI(uri).
		SetMonitor(monitor).
		SetPoolMonitor(poolMonitor)

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
		err := mongodb.EnsureSchema(database, lifecycle)
		if err != nil {
			return nil, fmt.Errorf("failed to ensure mongo schema: %w", err)
		}
	}

	return database, nil
}

func getObservabilityForMongo(
	meterProvider metricapi.MeterProvider,
	traceProvider traceapi.TracerProvider,
	lifecycle fx.Lifecycle,
) (*event.CommandMonitor, *event.PoolMonitor, error) {
	monitor := otelmongo.NewMonitor(
		otelmongo.WithMeterProvider(meterProvider),
		otelmongo.WithTracerProvider(traceProvider),
	)
	meterCtx, meterCancel := context.WithCancel(context.Background())
	lifecycle.Append(fx.Hook{
		OnStart: nil,
		OnStop: func(context.Context) error {
			meterCancel()

			return nil
		},
	})

	poolMonitor, err := otelmongo.NewPoolMonitor(
		meterCtx,
		"apiserver-mongo-pool",
		otelmongo.WithPoolMeterProvider(meterProvider),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create mongo pool monitor: %w", err)
	}

	return monitor, poolMonitor, nil
}
