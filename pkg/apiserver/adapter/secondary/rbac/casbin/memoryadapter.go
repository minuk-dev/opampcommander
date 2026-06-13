package casbin

import (
	"log/slog"

	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
)

var _ persist.Adapter = (*noopAdapter)(nil)

// noopAdapter is a Casbin storage adapter that persists nothing. It exists so a
// purely in-memory enforcer (standalone mode) can still satisfy the enforcer's
// LoadPolicy/SavePolicy contract: the plain in-memory enforcer created from a
// model alone has a nil adapter, and calling SavePolicy on it panics.
//
// Policy rules live only in the enforcer's in-memory model; this adapter just
// makes load/save/add/remove succeed as no-ops, so RBAC sync works without any
// external policy store.
type noopAdapter struct{}

// LoadPolicy implements persist.Adapter. There is nothing to load.
func (*noopAdapter) LoadPolicy(model.Model) error { return nil }

// SavePolicy implements persist.Adapter. There is nothing to persist.
func (*noopAdapter) SavePolicy(model.Model) error { return nil }

// AddPolicy implements persist.Adapter (auto-save). No-op.
func (*noopAdapter) AddPolicy(string, string, []string) error { return nil }

// RemovePolicy implements persist.Adapter (auto-save). No-op.
func (*noopAdapter) RemovePolicy(string, string, []string) error { return nil }

// RemoveFilteredPolicy implements persist.Adapter (auto-save). No-op.
func (*noopAdapter) RemoveFilteredPolicy(string, string, int, ...string) error { return nil }

// NewInMemoryEnforcer creates an Enforcer whose policies live only in process
// memory, backed by a no-op storage adapter so SavePolicy/LoadPolicy are safe.
// Used in standalone mode where there is no external policy store.
func NewInMemoryEnforcer(logger *slog.Logger, casbinModel model.Model) (*Enforcer, error) {
	return NewEnforcerWithAdapter(logger, casbinModel, &noopAdapter{})
}
