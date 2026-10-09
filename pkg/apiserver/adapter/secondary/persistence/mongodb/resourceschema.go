package mongodb

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var errResourceSchemaNotReady = errors.New("resource schema is not ready")

// ValidateResourceSchema checks the write prerequisites without changing data or indexes.
func ValidateResourceSchema(ctx context.Context, database *mongo.Database) error {
	for _, plan := range fleetIndexes() {
		if !slices.Contains(versionedCollections, plan.collectionName) {
			continue
		}

		collection := database.Collection(plan.collectionName)

		err := validateResourceKeyIndex(ctx, collection, plan.indexes[0].Keys)
		if err != nil {
			return err
		}

		err = collection.FindOne(ctx, bson.M{
			resourceVersionFieldName: bson.M{"$not": bson.M{"$gt": int64(0)}},
		}).Err()
		if err == nil {
			return fmt.Errorf("%w: %s contains unmigrated resource versions", errResourceSchemaNotReady, plan.collectionName)
		}

		if !errors.Is(err, mongo.ErrNoDocuments) {
			return fmt.Errorf("validate resource versions in %s: %w", plan.collectionName, err)
		}
	}

	return nil
}

func validateResourceKeyIndex(ctx context.Context, collection *mongo.Collection, keys any) error {
	required, err := bson.Marshal(keys)
	if err != nil {
		return fmt.Errorf("encode required resource index: %w", err)
	}

	cursor, err := collection.Indexes().List(ctx)
	if err != nil {
		return fmt.Errorf("list resource indexes in %s: %w", collection.Name(), err)
	}

	var indexes []struct {
		Key     bson.Raw `bson:"key"`
		Unique  bool     `bson:"unique"`
		Sparse  bool     `bson:"sparse"`
		Partial bson.Raw `bson:"partialFilterExpression"`
	}

	err = cursor.All(ctx, &indexes)
	if err != nil {
		return fmt.Errorf("decode resource indexes in %s: %w", collection.Name(), err)
	}

	for _, index := range indexes {
		if index.Unique && !index.Sparse && len(index.Partial) == 0 && bytes.Equal(index.Key, required) {
			return nil
		}
	}

	return fmt.Errorf("%w: %s is missing its full unique resource-key index", errResourceSchemaNotReady, collection.Name())
}
