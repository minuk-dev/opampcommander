// Package mongodb provides the MongoDB adapter for the opampcommander application.
package mongodb

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/fx"
)

func InitialMongoDBCollection(
	database *mongo.Database,
	lifecycle fx.Lifecycle,
) error {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			collections := []string{
				agentCollectionName,
				agentGroupCollectionName,
			}
			err := createNonExistingCollections(ctx, database, collections)
			if err != nil {
				return fmt.Errorf("failed to create non-existing collections: %w", err)
			}

			return nil
		},
		OnStop: nil,
	})

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
			return fmt.Errorf("failed to create collection %s: %w", collectionName, err)
		}
	}

	return nil
}

func createNonExistingIndexes(
	ctx context.Context,
	database *mongo.Database,
	collectionName string,
	indexes []mongo.IndexModel,
) error {
	collection := database.Collection(collectionName)

	existingIndexesCursor, err := collection.Indexes().List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list existing indexes: %w", err)
	}
	defer existingIndexesCursor.Close(ctx)

	existingIndexes := make(map[string]struct{})
	for existingIndexesCursor.Next(ctx) {
		var indexInfo bson.M
		if err := existingIndexesCursor.Decode(&indexInfo); err != nil {
			return fmt.Errorf("failed to decode index info: %w", err)
		}
		if name, ok := indexInfo["name"].(string); ok {
			existingIndexes[name] = struct{}{}
		}
	}

	var indexesToCreate []mongo.IndexModel
	for _, index := range indexes {
		indexName, err := index.Options.Name, error(nil)
		if indexName == nil {
			indexName, err = mongo.IndexName(index.Keys)
			if err != nil {
				return fmt.Errorf("failed to generate index name: %w", err)
			}
			index.Options.Name = &indexName
		}

		if _, exists := existingIndexes[*indexName]; !exists {
			indexesToCreate = append(indexesToCreate, index)
		}
	}

	if len(indexesToCreate) > 0 {
		_, err := collection.Indexes().CreateMany(ctx, indexesToCreate)
		if err != nil {
			return fmt.Errorf("failed to create indexes: %w", err)
		}
	}

	return nil
}
