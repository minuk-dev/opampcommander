package connection

import (
	"time"

	"github.com/google/uuid"
)

// Connection represents a connection to an agent.
type Connection struct {
	ID                 uuid.UUID `json:"id"`
	InstanceUID        uuid.UUID `json:"instanceUid"`
	LastCommunicatedAt time.Time `json:"lastCommunicatedAt"`
	Alive              bool      `json:"alive"`
}
