package apiserver

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	metricapi "go.opentelemetry.io/otel/metric"
	otelpropagation "go.opentelemetry.io/otel/propagation"
	traceapi "go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	experimental "google.golang.org/grpc/experimental/opentelemetry"
	"google.golang.org/grpc/stats/opentelemetry"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/config"
)

// Controller is an interface that defines the methods for handling HTTP requests.
type Controller interface {
	RoutesInfo() gin.RoutesInfo
}

// NewEtcdClient creates a new etcd client with the given settings.
func NewEtcdClient(
	settings *config.ServerSettings,
	meterProvider metricapi.MeterProvider,
	traceProvider traceapi.TracerProvider,
	textMapPropagator otelpropagation.TextMapPropagator,
	lifecycle fx.Lifecycle,
) (*clientv3.Client, error) {
	observabilityDialOpt := opentelemetry.DialOption(opentelemetry.Options{
		MetricsOptions: opentelemetry.MetricsOptions{
			MeterProvider:         meterProvider,
			Metrics:               opentelemetry.DefaultMetrics(),
			MethodAttributeFilter: nil,
			OptionalLabels:        nil,
		},
		TraceOptions: experimental.TraceOptions{
			TracerProvider:    traceProvider,
			TextMapPropagator: textMapPropagator,
		},
	})
	//exhaustruct:ignore
	etcdConfig := clientv3.Config{
		Endpoints: settings.DatabaseSettings.Endpoints,
		DialOptions: []grpc.DialOption{
			observabilityDialOpt,
		},
	}

	etcdClient, err := clientv3.New(etcdConfig)
	if err != nil {
		return nil, fmt.Errorf("etcd client init failed: %w", err)
	}

	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error { return nil },
		OnStop: func(_ context.Context) error {
			err := etcdClient.Close()
			if err != nil {
				return fmt.Errorf("failed to close etcd client: %w", err)
			}

			return nil
		},
	})

	return etcdClient, nil
}

// NewMongoDBClient creates a new MongoDB client with the given settings.
func NewMongoDBClient(
	settings *config.ServerSettings,
	meterProvider metricapi.MeterProvider,
	traceProvider traceapi.TracerProvider,
	textMapPropagator otelpropagation.TextMapPropagator,
	lifecycle fx.Lifecycle,
) (*mongo.Client, error) {
	timeout := settings.DatabaseSettings.ConnectTimeout
	if timeout == 0 {
		timeout = 10 * 1000000000 // 10 seconds in nanoseconds
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		timeout,
	)
	defer cancel()

	// Use the first endpoint as the connection string
	var uri string
	if len(settings.DatabaseSettings.Endpoints) > 0 {
		uri = settings.DatabaseSettings.Endpoints[0]
	} else {
		uri = "mongodb://localhost:27017"
	}

	mongoClient, err := mongo.Connect(
		ctx,
		options.Client().ApplyURI(uri),
	)
	if err != nil {
		return nil, fmt.Errorf("mongo client init failed: %w", err)
	}

	// Ping to verify connection
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
	settings *config.ServerSettings,
) *mongo.Database {
	// Use a default database name if not specified
	// You can extract database name from connection string or use a default
	databaseName := "opampcommander"
	
	return client.Database(databaseName)
}
