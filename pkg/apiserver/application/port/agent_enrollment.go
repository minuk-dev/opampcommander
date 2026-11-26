package port

import (
	"context"
	"errors"
)

// ErrAgentEnrollmentConflict means the Agent identity already belongs to another ServiceAccount.
var ErrAgentEnrollmentConflict = errors.New("agent enrollment belongs to another service account")

// AgentEnrollmentKey identifies an Agent within a specific incarnation of its namespace.
type AgentEnrollmentKey struct {
	InstallationUID string
	NamespaceUID    string
	InstanceUID     string
}

// AgentEnrollmentStore persists immutable Agent credential bindings.
type AgentEnrollmentStore interface {
	Claim(ctx context.Context, key AgentEnrollmentKey, serviceAccountUID string) error
}
