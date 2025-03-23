package agent

import "github.com/google/uuid"

type Agent struct {
	InstanceUID uuid.UUID `json:"instanceUid"`
	Raw         any       `json:"raw"`
}
