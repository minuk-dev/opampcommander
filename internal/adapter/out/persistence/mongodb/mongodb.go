// Package mongodb provides the MongoDB adapter for the opampcommander application.
package mongodb

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

//nolint:gochecknoglobals // These are constants for collection names and indexes.
var (
	collections = []string{
		agentCollectionName,
		agentGroupCollectionName,
		serverCollectionName,
	}

	indexes = []collectionAndIndexes{
		{
			collectionName: agentCollectionName,
			indexes: []mongo.IndexModel{
				{
					Keys: bson.D{
						{Key: "metadata.instanceUid", Value: 1},
					},
					Options: nil,
				},
				{
					Keys: bson.D{
						{Key: "metadata.description.identifyingAttributes.key", Value: 1},
						{Key: "metadata.description.identifyingAttributes.value", Value: 1},
					},
					Options: nil,
				},
				{
					Keys: bson.D{
						{Key: "metadata.description.nonIdentifyingAttributes.key", Value: 1},
						{Key: "metadata.description.nonIdentifyingAttributes.value", Value: 1},
					},
					Options: nil,
				},
				// Index for status.connected field - used in AgentGroup statistics aggregation
				{
					Keys: bson.D{
						{Key: "status.connected", Value: 1},
					},
					Options: nil,
				},
				// Index for status.componentHealth.healthy field - used in AgentGroup statistics aggregation
				{
					Keys: bson.D{
						{Key: "status.componentHealth.healthy", Value: 1},
					},
					Options: nil,
				},
			},
		},
		{
			collectionName: agentGroupCollectionName,
			indexes: []mongo.IndexModel{
				{
					Keys: bson.D{
						{Key: "name", Value: 1},
					},
					Options: nil,
				},
			},
		},
		{
			collectionName: serverCollectionName,
			indexes: []mongo.IndexModel{
				{
					Keys: bson.D{
						{Key: "serverId", Value: 1},
					},
					Options: nil,
				},
			},
		},
	}
)

// EnsureSchema ensures that the necessary collections and indexes exist in the MongoDB database.
// This function should be called during application startup.
func EnsureSchema(
	ctx context.Context,
	database *mongo.Database,
) error {
	err := createNonExistingCollections(ctx, database, collections)
	if err != nil {
		return fmt.Errorf("failed to create non-existing collections: %w", err)
	}

	err = createIndexes(ctx, database, indexes)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}

func createNonExistingCollections(
	ctx context.Context,
	database *mongo.Database,
	collections []string,
) error {
	existingCollections, err := database.ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return fmt.Errorf("failed to list existing collections: %w", err)
	}

	notExistingCollections := lo.Filter(collections, func(c string, _ int) bool {
		return !lo.Contains(existingCollections, c)
	})

	for _, collectionName := range notExistingCollections {
		err := database.CreateCollection(ctx, collectionName)
		if err != nil {
			// Ignore NamespaceExists error (code 48) which can occur in distributed setups
			// when multiple servers try to create the same collection concurrently
			var cmdErr mongo.CommandError
			if errors.As(err, &cmdErr) && cmdErr.Code == 48 { // NamespaceExists
				continue
			}

			return fmt.Errorf("failed to create collection %s: %w", collectionName, err)
		}
	}

	return nil
}

type collectionAndIndexes struct {
	collectionName string
	indexes        []mongo.IndexModel
}

func createIndexes(
	ctx context.Context,
	database *mongo.Database,
	indexes []collectionAndIndexes,
) error {
	for _, ci := range indexes {
		collection := database.Collection(ci.collectionName)

		_, err := collection.Indexes().CreateMany(ctx, ci.indexes)
		if err != nil {
			return fmt.Errorf("failed to create indexes for collection %s: %w", ci.collectionName, err)
		}
	}

	return nil
}
