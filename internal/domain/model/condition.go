package model

import "time"

// Condition represents a condition of a resource.
type Condition struct {
	// Type is the type of the condition.
	Type ConditionType
	// LastTransitionTime is the last time the condition transitioned.
	LastTransitionTime time.Time
	// Status is the status of the condition.
	Status ConditionStatus
	// Reason is the identifier of the user or system that triggered the condition.
	Reason string
	// Message is a human readable message indicating details about the condition.
	Message string
}

// ConditionType represents the type of a condition.
type ConditionType string

const (
	// ConditionTypeCreated represents the condition when the resource was created.
	ConditionTypeCreated ConditionType = "Created"
	// ConditionTypeUpdated represents the condition when the resource was updated.
	ConditionTypeUpdated ConditionType = "Updated"
	// ConditionTypeDeleted represents the condition when the resource was deleted.
	ConditionTypeDeleted ConditionType = "Deleted"
	// ConditionTypeAlive represents the condition when the server is alive.
	ConditionTypeAlive ConditionType = "Alive"
)

// ConditionStatus represents the status of a condition.
type ConditionStatus string

const (
	// ConditionStatusTrue represents a true condition status.
	ConditionStatusTrue ConditionStatus = "True"
	// ConditionStatusFalse represents a false condition status.
	ConditionStatusFalse ConditionStatus = "False"
	// ConditionStatusUnknown represents an unknown condition status.
	ConditionStatusUnknown ConditionStatus = "Unknown"
)
