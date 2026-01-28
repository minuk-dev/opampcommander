package entity

import (
	"go.mongodb.org/mongo-driver/v2/bson"

	domainmodel "github.com/minuk-dev/opampcommander/internal/domain/model"
)

// Condition represents a condition of an agent group in MongoDB.
type Condition struct {
	Type               string        `bson:"type"`
	LastTransitionTime bson.DateTime `bson:"lastTransitionTime"`
	Status             string        `bson:"status"`
	Reason             string        `bson:"reason"`
	Message            string        `bson:"message,omitempty"`
}

// NewConditionFromDomain creates a new Condition from domain model.
func NewConditionFromDomain(condition domainmodel.Condition) Condition {
	return Condition{
		Type:               string(condition.Type),
		Status:             string(condition.Status),
		LastTransitionTime: bson.NewDateTimeFromTime(condition.LastTransitionTime),
		Reason:             condition.Reason,
		Message:            condition.Message,
	}
}

// ToDomain converts Condition to domain model.
func (c *Condition) ToDomain() domainmodel.Condition {
	return domainmodel.Condition{
		Type:               domainmodel.ConditionType(c.Type),
		Status:             domainmodel.ConditionStatus(c.Status),
		LastTransitionTime: c.LastTransitionTime.Time(),
		Reason:             c.Reason,
		Message:            c.Message,
	}
}
