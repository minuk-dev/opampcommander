package inmemory

import (
	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent/model"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/port"
)

// namespacedName is the composite key for resources identified by a
// (namespace, name) pair, e.g. agent groups, packages, remote configs,
// certificates, and role bindings.
type namespacedName struct {
	Namespace string
	Name      string
}

// errResourceNotExist returns the shared not-found error so callers (and the
// HTTP layer's RFC 9457 mapping) treat in-memory misses exactly like MongoDB's.
func errResourceNotExist() error {
	return port.ErrResourceNotExist
}

// errConflict returns the shared optimistic-concurrency error so the in-memory
// store rejects stale writes exactly like the MongoDB adapter's version check.
func errConflict() error {
	return port.ErrConflict
}

// matchesSelector reports whether the agent satisfies every identifying and
// non-identifying attribute in the selector. An empty selector matches all
// agents, mirroring the MongoDB selector-to-filter behaviour.
func matchesSelector(agent *agentmodel.Agent, selector agentmodel.AgentSelector) bool {
	for key, value := range selector.IdentifyingAttributes {
		if agent.Metadata.Description.IdentifyingAttributes[key] != value {
			return false
		}
	}

	for key, value := range selector.NonIdentifyingAttributes {
		if agent.Metadata.Description.NonIdentifyingAttributes[key] != value {
			return false
		}
	}

	return true
}

// matchesAttributes reports whether the stored attribute map contains every
// key=value pair in the selector (an AND of equality conditions). An empty
// selector matches everything, mirroring the MongoDB attribute filter used by the
// namespaced agent listing. It is used for both identifying and non-identifying
// attributes.
func matchesAttributes(stored, selector map[string]string) bool {
	for key, value := range selector {
		if stored[key] != value {
			return false
		}
	}

	return true
}
