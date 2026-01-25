package entity

import (
	"time"

	domainmodel "github.com/minuk-dev/opampcommander/internal/domain/model"
)

// Condition represents a condition of an agent group in MongoDB.
type Condition struct {
	Type               string    `bson:"type"`
	LastTransitionTime time.Time `bson:"lastTransitionTime"`
	Status             string    `bson:"status"`
	Reason             string    `bson:"reason"`
	Message            string    `bson:"message,omitempty"`
}

func NewConnectionFromDomain(c domainmodel.Condition) Condition {
	return Condition{
		Type:               string(c.Type),
		Status:             string(c.Status),
		LastTransitionTime: c.LastTransitionTime,
		Reason:             c.Reason,
		Message:            c.Message,
	}
}

func (c *Condition) ToDomain() domainmodel.Condition {
	return domainmodel.Condition{
		Type:               domainmodel.ConditionType(c.Type),
		Status:             domainmodel.ConditionStatus(c.Status),
		LastTransitionTime: c.LastTransitionTime,
		Reason:             c.Reason,
		Message:            c.Message,
	}
}
