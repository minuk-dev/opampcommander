// Package port is a package that defines the ports for the application layer.
package port

import (
	"errors"

	agentport "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/port"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
)

// ErrResourceNotExist is returned when a requested resource does not exist. It aliases the
// domain sentinel so primary adapters can map it (e.g. to a 404) without importing the domain.
var ErrResourceNotExist = model.ErrResourceNotExist

// ErrAgentConnected is returned when attempting to delete an agent that is still connected.
// Only disconnected agents can be deleted. It aliases the domain sentinel (the guard is
// enforced in the domain layer) so the HTTP layer can match it without importing the domain.
var ErrAgentConnected = agentport.ErrAgentConnected

// ErrAgentNamespaceMismatch is returned when an agent exists but does not belong to the
// requested namespace. From that namespace's perspective the agent does not exist, so
// callers should map this to a 404.
var ErrAgentNamespaceMismatch = errors.New("agent does not belong to the specified namespace")
