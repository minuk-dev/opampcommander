package infrastructure

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/minuk-dev/opampcommander/internal/management/healthcheck"
)

// EnsureSchema ensures that the necessary collections and indexes exist in the MongoDB database.
var _ healthcheck.HealthIndicator = (*MongoDBHealthIndicator)(nil)

// MongoDBHealthIndicator is a health indicator for MongoDB.
type MongoDBHealthIndicator struct {
	client *mongo.Client
}

// NewMongoDBHealthIndicator creates a new MongoDBHealthIndicator.
func NewMongoDBHealthIndicator(client *mongo.Client) *MongoDBHealthIndicator {
	return &MongoDBHealthIndicator{
		client: client,
	}
}

// Name returns the name of the health indicator.
func (m *MongoDBHealthIndicator) Name() string {
	return "MongoDB"
}

// Readiness returns the readiness status of the MongoDB connection.
func (m *MongoDBHealthIndicator) Readiness(ctx context.Context) healthcheck.Readiness {
	err := m.client.Ping(ctx, nil)
	if err != nil {
		return healthcheck.Readiness{
			Ready:  false,
			Reason: err.Error(),
		}
	}

	return healthcheck.Readiness{
		Ready:  true,
		Reason: "",
	}
}

// Health returns the health status of the MongoDB connection.
func (m *MongoDBHealthIndicator) Health(context.Context) healthcheck.Health {
	return healthcheck.Health{
		Healthy: true,
		Reason:  "when mongodb is initialized, it's always healthy",
	}
}
