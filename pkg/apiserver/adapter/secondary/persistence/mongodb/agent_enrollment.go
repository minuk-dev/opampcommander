package mongodb

import (
	"context"
	"errors"
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
	update := bson.M{"$setOnInsert": bson.M{"serviceAccountUID": serviceAccountUID}}

	_, err := s.collection.UpdateOne(ctx, enrollmentFilter(key), update, options.UpdateOne().SetUpsert(true))
	if err != nil && !mongo.IsDuplicateKeyError(err) {
		return fmt.Errorf("claim agent enrollment: %w", err)
	}

	owner, err := s.Lookup(ctx, key)
	if err != nil {
		return fmt.Errorf("read agent enrollment owner: %w", err)
	}

	if owner != serviceAccountUID {
		return port.ErrAgentEnrollmentConflict
	}

	return nil
}

// Lookup reads the majority-committed owner of this Agent identity.
func (s *AgentEnrollmentStore) Lookup(ctx context.Context, key port.AgentEnrollmentKey) (string, error) {
	var stored struct {
		ServiceAccountUID string `bson:"serviceAccountUID"`
	}

	err := s.collection.FindOne(ctx, enrollmentFilter(key)).Decode(&stored)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return "", port.ErrAgentEnrollmentNotFound
	}

	if err != nil {
		return "", fmt.Errorf("read agent enrollment: %w", err)
	}

	return stored.ServiceAccountUID, nil
}

func enrollmentFilter(key port.AgentEnrollmentKey) bson.M {
	return bson.M{"_id": bson.D{
		{Key: "installationUID", Value: key.InstallationUID},
		{Key: "namespaceUID", Value: key.NamespaceUID},
		{Key: "instanceUID", Value: key.InstanceUID},
	}}
}
