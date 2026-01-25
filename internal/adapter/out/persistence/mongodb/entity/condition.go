package entity

import "time"

// Condition represents a condition of an agent group in MongoDB.
type Condition struct {
	Type               string    `bson:"type"`
	LastTransitionTime time.Time `bson:"lastTransitionTime"`
	Status             string    `bson:"status"`
	Reason             string    `bson:"reason"`
	Message            string    `bson:"message,omitempty"`
}
