package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/minuk-dev/opampcommander/pkg/apiserver/application/port"
)

// AgentEnrollmentStore stores Agent enrollment in the installation's MongoDB database.
type AgentEnrollmentStore struct {
	collection *mongo.Collection
}

// NewAgentEnrollmentStore uses majority reads and writes for credential ownership.
func NewAgentEnrollmentStore(database *mongo.Database) *AgentEnrollmentStore {
	return &AgentEnrollmentStore{collection: runtimeMetadata(database)}
}

// Claim records the first verified ServiceAccount UID and makes retries by that identity idempotent.
func (s *AgentEnrollmentStore) Claim(ctx context.Context, key port.AgentEnrollmentKey, serviceAccountUID string) error {
	filter := bson.M{"_id": bson.D{
		{Key: "installationUID", Value: key.InstallationUID},
		{Key: "namespaceUID", Value: key.NamespaceUID},
		{Key: "instanceUID", Value: key.InstanceUID},
	}}
	update := bson.M{"$setOnInsert": bson.M{"serviceAccountUID": serviceAccountUID}}

	_, err := s.collection.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true))
	if err != nil && !mongo.IsDuplicateKeyError(err) {
		return fmt.Errorf("claim agent enrollment: %w", err)
	}

	var stored struct {
		ServiceAccountUID string `bson:"serviceAccountUID"`
	}

	err = s.collection.FindOne(ctx, filter).Decode(&stored)
	if err != nil {
		return fmt.Errorf("read agent enrollment owner: %w", err)
	}

	if stored.ServiceAccountUID != serviceAccountUID {
		return port.ErrAgentEnrollmentConflict
	}

	return nil
}
