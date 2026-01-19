package v1

// Condition represents a condition of an agent group.
type Condition struct {
	Type               ConditionType   `json:"type"`
	LastTransitionTime Time            `json:"lastTransitionTime"`
	Status             ConditionStatus `json:"status"`
	Reason             string          `json:"reason"`
	Message            string          `json:"message,omitempty"`
} // @name Condition

// ConditionType represents the type of an agent condition.
type ConditionType string // @name ConditionType

const (
	// ConditionTypeCreated represents the condition when the agent group was created.
	ConditionTypeCreated ConditionType = "Created"
	// ConditionTypeUpdated represents the condition when the agent group was updated.
	ConditionTypeUpdated ConditionType = "Updated"
	// ConditionTypeDeleted represents the condition when the agent group was deleted.
	ConditionTypeDeleted ConditionType = "Deleted"
	// ConditionTypeConnected represents the condition when the agent is connected.
	ConditionTypeConnected ConditionType = "Connected"
	// ConditionTypeHealthy represents the condition when the agent is healthy.
	ConditionTypeHealthy ConditionType = "Healthy"
	// ConditionTypeConfigured represents the condition when the agent has been configured.
	ConditionTypeConfigured ConditionType = "Configured"
	// ConditionTypeRegistered represents the condition when the agent has been registered.
	ConditionTypeRegistered ConditionType = "Registered"
)

// ConditionStatus represents the status of an agent condition.
type ConditionStatus string // @name AgentConditionStatus

const (
	// ConditionStatusTrue represents a true condition status.
	ConditionStatusTrue ConditionStatus = "True"
	// ConditionStatusFalse represents a false condition status.
	ConditionStatusFalse ConditionStatus = "False"
	// ConditionStatusUnknown represents an unknown condition status.
	ConditionStatusUnknown ConditionStatus = "Unknown"
)
