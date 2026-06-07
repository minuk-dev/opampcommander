// Package entity contains common entity definitions for MongoDB persistence.
package entity

import "go.mongodb.org/mongo-driver/v2/bson"

const (
	// VersionV1 is the initial version of the agent group.
	VersionV1 Version = 1
)

// Version represents the version of the entity's schema.
type Version int

// Common is a struct that contains common fields for all entities.
// All entities SHOULD embed this struct.
type Common struct {
	// Version is the version of the entity's schema.
	Version Version `bson:"version"`

	// ID is the unique identifier of the entity.
	// It is used as the primary key in the database.
	// IT is used internally and as a continueToken for pagination.
	ID *bson.ObjectID `bson:"_id,omitempty"`
}
