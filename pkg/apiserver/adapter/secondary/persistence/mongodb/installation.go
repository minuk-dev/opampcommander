package mongodb

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

var errInstallationMismatch = errors.New("MongoDB installation binding does not match configured installation UID")

// VerifyInstallation checks the database owner before schema migration or runtime startup.
// With automatic DDL, competing gateways can initialize the singleton but cannot replace it.
func VerifyInstallation(ctx context.Context, database *mongo.Database, uid string, ddlAuto bool) error {
	collection := database.Collection("runtime_metadata", options.Collection().
		SetReadPreference(readpref.Primary()).
		SetReadConcern(readconcern.Majority()).
		SetWriteConcern(writeconcern.Majority()))
	filter := bson.M{"_id": "installation"}

	if uid == "" {
		return errInstallationMismatch
	}

	if ddlAuto {
		update := bson.M{"$setOnInsert": bson.M{"uid": uid}}

		_, err := collection.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true))
		if err != nil && !mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("initialize MongoDB installation binding: %w", err)
		}
	}

	var stored struct {
		UID string `bson:"uid"`
	}

	err := collection.FindOne(ctx, filter).Decode(&stored)
	if err != nil {
		return fmt.Errorf("read MongoDB installation binding: %w", err)
	}

	if stored.UID != uid {
		return errInstallationMismatch
	}

	return nil
}
