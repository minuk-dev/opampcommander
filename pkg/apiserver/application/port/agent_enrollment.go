package port

import (
	"context"
	"errors"
)

// ErrAgentEnrollmentConflict means the Agent identity already belongs to another ServiceAccount.
var ErrAgentEnrollmentConflict = errors.New("agent enrollment belongs to another service account")

// ErrAgentEnrollmentNotFound means no binding exists for the specified Agent identity.
var ErrAgentEnrollmentNotFound = errors.New("agent enrollment not found")

// AgentEnrollmentKey identifies an Agent within a specific incarnation of its namespace.
type AgentEnrollmentKey struct {
	InstallationUID string
	NamespaceUID    string
	InstanceUID     string
}

// AgentEnrollmentStore persists immutable Agent credential bindings.
type AgentEnrollmentStore interface {
	Claim(ctx context.Context, key AgentEnrollmentKey, serviceAccountUID string) error
	Lookup(ctx context.Context, key AgentEnrollmentKey) (serviceAccountUID string, err error)
}
