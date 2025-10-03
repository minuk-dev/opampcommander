package entity

import "go.mongodb.org/mongo-driver/bson/primitive"

const (
	// Version1 is the initial version of the agent group.
	Version1 Version = 1
)

type Version int

// EntityCommon is a struct that contains common fields for all entities.
// All entities SHOULD embed this struct.
type EntityCommon struct {
	// Version is the version of the entity's schema.
	Version Version `bson:"version"`

	// ID is the unique identifier of the entity.
	// It is used as the primary key in the database.
	// IT is used internally and as a continueToken for pagination.
	ID *primitive.ObjectID `bson:"_id,omitempty"`
}
