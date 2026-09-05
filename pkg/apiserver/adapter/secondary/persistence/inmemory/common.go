package inmemory

import (
	"fmt"

	agentmodel "github.com/minuk-dev/opampcommander/pkg/apiserver/domain/agent"
	"github.com/minuk-dev/opampcommander/pkg/apiserver/domain/model"
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
	return model.ErrResourceNotExist
}

// errConflict returns the shared optimistic-concurrency error so the in-memory
// store rejects stale writes exactly like the MongoDB adapter's version check.
func errConflict() error {
	return model.ErrConflict
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

// ErrSelectorUnsupported is returned when a listing carries a label or field
// selector for a resource this adapter holds no selector projection for. It
// wraps model.ErrInvalidArgument so it surfaces as a 400, exactly as the same
// situation does on the MongoDB side.
var ErrSelectorUnsupported = fmt.Errorf(
	"%w: this resource does not support selector filtering", model.ErrInvalidArgument)

// ErrLabelsUnsupported is returned when a listing carries a label selector for a
// resource that has no label map at all — a role, say. It is an error rather than
// an empty page so the answer matches the MongoDB adapter's and the API
// boundary's, and so a client is told why nothing came back.
var ErrLabelsUnsupported = fmt.Errorf(
	"%w: this resource has no labels to select on", model.ErrInvalidArgument)

// namespaceFilter restricts a listing to one namespace, or to every namespace
// when it is empty. It is the in-memory counterpart of the MongoDB adapters'
// namespace match condition.
func namespaceFilter[T any](namespace string, namespaceOf func(T) string) func(T) bool {
	if namespace == "" {
		return nil
	}

	return func(value T) bool {
		return namespaceOf(value) == namespace
	}
}
