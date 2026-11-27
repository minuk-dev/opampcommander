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

const agentEnrollmentCollectionName = "agent_enrollments"

// AgentEnrollmentStore stores Agent enrollment in the installation's MongoDB database.
type AgentEnrollmentStore struct {
	collection *mongo.Collection
	legacy     *mongo.Collection
}

// NewAgentEnrollmentStore uses majority reads and writes for credential ownership.
func NewAgentEnrollmentStore(database *mongo.Database) *AgentEnrollmentStore {
	return &AgentEnrollmentStore{
		collection: runtimeCollection(database, agentEnrollmentCollectionName),
		legacy:     runtimeCollection(database, runtimeMetadataCollectionName),
	}
}

// Claim records the first verified ServiceAccount UID and makes retries by that identity idempotent.
func (s *AgentEnrollmentStore) Claim(ctx context.Context, key port.AgentEnrollmentKey, serviceAccountUID string) error {
	owner, err := s.Lookup(ctx, key)
	if err == nil {
		if owner != serviceAccountUID {
			return port.ErrAgentEnrollmentConflict
		}

		return nil
	}

	if !errors.Is(err, port.ErrAgentEnrollmentNotFound) {
		return fmt.Errorf("check current agent enrollment owner: %w", err)
	}

	update := bson.M{"$setOnInsert": bson.M{"serviceAccountUID": serviceAccountUID}}

	_, err = s.collection.UpdateOne(ctx, enrollmentFilter(key), update, options.UpdateOne().SetUpsert(true))
	if err != nil && !mongo.IsDuplicateKeyError(err) {
		return fmt.Errorf("claim agent enrollment: %w", err)
	}

	owner, err = lookupEnrollment(ctx, s.collection, key)
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
	owner, err := lookupEnrollment(ctx, s.collection, key)
	if errors.Is(err, port.ErrAgentEnrollmentNotFound) {
		// Existing bindings remain authoritative until an explicit migration.
		return lookupEnrollment(ctx, s.legacy, key)
	}

	return owner, err
}

func lookupEnrollment(ctx context.Context, collection *mongo.Collection, key port.AgentEnrollmentKey) (string, error) {
	var stored struct {
		ServiceAccountUID string `bson:"serviceAccountUID"`
	}

	err := collection.FindOne(ctx, enrollmentFilter(key)).Decode(&stored)
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
