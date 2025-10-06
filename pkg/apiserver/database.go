package apiserver

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	metricapi "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
)

// Controller is an interface that defines the methods for handling HTTP requests.
type Controller interface {
	RoutesInfo() gin.RoutesInfo
}

// NewMongoDBClient creates a new MongoDB client with the given settings.
func NewMongoDBClient(
	settings *config.ServerSettings,
	_ metricapi.MeterProvider,
	traceProvider trace.TracerProvider,
	lifecycle fx.Lifecycle,
) (*mongo.Client, error) {
	const defaultTimeout = 10 * time.Second

	timeout := settings.DatabaseSettings.ConnectTimeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var uri string
	if len(settings.DatabaseSettings.Endpoints) > 0 {
		uri = settings.DatabaseSettings.Endpoints[0]
	} else {
		uri = "mongodb://localhost:27017"
	}

	tracer := traceProvider.Tracer("mongodb")

	// Setup command monitoring for observability
	cmdMonitor := &event.CommandMonitor{
		Started: func(ctx context.Context, evt *event.CommandStartedEvent) {
			_, span := tracer.Start(ctx, "mongodb."+evt.CommandName)
			defer span.End()
		},
		Succeeded: func(_ context.Context, _ *event.CommandSucceededEvent) {},
		Failed:    func(_ context.Context, _ *event.CommandFailedEvent) {},
	}

	clientOptions := options.Client().ApplyURI(uri).SetMonitor(cmdMonitor)

	mongoClient, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("mongo client init failed: %w", err)
	}

	err = mongoClient.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("mongo client ping failed: %w", err)
	}

	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error { return nil },
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
	_ *config.ServerSettings,
) *mongo.Database {
	// Use a default database name if not specified
	// You can extract database name from connection string or use a default
	databaseName := "opampcommander"

	return client.Database(databaseName)
}
