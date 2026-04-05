// Package casbin provides a Casbin-based RBAC enforcer adapter
// that implements the RBACEnforcerPort interface for permission management.
package casbin

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"

	userport "github.com/minuk-dev/opampcommander/internal/domain/user/port"
)

var _ userport.RBACEnforcerPort = (*Enforcer)(nil)

// Enforcer wraps a Casbin enforcer to implement RBACEnforcerPort.
type Enforcer struct {
	enforcer *casbin.Enforcer
	logger   *slog.Logger
}

// NewEnforcer creates a new Enforcer using a model config file.
// Policies are stored in-memory only.
func NewEnforcer(logger *slog.Logger, modelPath string) (*Enforcer, error) {
	casbinEnforcer, err := casbin.NewEnforcer(modelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin enforcer: %w", err)
	}

	return &Enforcer{enforcer: casbinEnforcer, logger: logger}, nil
}

// NewEnforcerFromModel creates a new Enforcer using a pre-loaded model.
// Policies are stored in-memory only.
func NewEnforcerFromModel(
	logger *slog.Logger,
	casbinModel model.Model,
) (*Enforcer, error) {
	casbinEnforcer, err := casbin.NewEnforcer(casbinModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin enforcer: %w", err)
	}

	return &Enforcer{enforcer: casbinEnforcer, logger: logger}, nil
}

// NewEnforcerWithAdapter creates a new Enforcer with a custom
// policy adapter (e.g., MongoDB) for persistent policy storage.
func NewEnforcerWithAdapter(
	logger *slog.Logger,
	casbinModel model.Model,
	adapter persist.Adapter,
) (*Enforcer, error) {
	casbinEnforcer, err := casbin.NewEnforcer(casbinModel, adapter)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create casbin enforcer with adapter: %w", err,
		)
	}

	return &Enforcer{enforcer: casbinEnforcer, logger: logger}, nil
}

// CheckPermission checks whether a user has the given permission on a resource in a namespace.
func (c *Enforcer) CheckPermission(_ context.Context, sub, dom, obj, act string) (bool, error) {
	allowed, err := c.enforcer.Enforce(sub, dom, obj, act)
	if err != nil {
		return false, fmt.Errorf("failed to check permission: %w", err)
	}

	c.logger.Debug("permission check",
		slog.String("sub", sub),
		slog.String("dom", dom),
		slog.String("obj", obj),
		slog.String("act", act),
		slog.Bool("allowed", allowed),
	)

	return allowed, nil
}

// LoadPolicy reloads all policies from the storage adapter.
func (c *Enforcer) LoadPolicy(_ context.Context) error {
	err := c.enforcer.LoadPolicy()
	if err != nil {
		return fmt.Errorf("failed to load policy: %w", err)
	}

	return nil
}

// SavePolicy persists all current policies to the storage adapter.
func (c *Enforcer) SavePolicy(_ context.Context) error {
	err := c.enforcer.SavePolicy()
	if err != nil {
		return fmt.Errorf("failed to save policy: %w", err)
	}

	return nil
}

// AddGroupingPolicy adds a role inheritance (grouping) policy rule.
func (c *Enforcer) AddGroupingPolicy(_ context.Context, params ...any) (bool, error) {
	added, err := c.enforcer.AddGroupingPolicy(params...)
	if err != nil {
		return false, fmt.Errorf("failed to add grouping policy: %w", err)
	}

	return added, nil
}

// RemoveGroupingPolicy removes a role inheritance (grouping) policy rule.
func (c *Enforcer) RemoveGroupingPolicy(_ context.Context, params ...any) (bool, error) {
	removed, err := c.enforcer.RemoveGroupingPolicy(params...)
	if err != nil {
		return false, fmt.Errorf("failed to remove grouping policy: %w", err)
	}

	return removed, nil
}

// GetGroupingPolicy returns all role inheritance (grouping) policy rules.
func (c *Enforcer) GetGroupingPolicy() ([][]string, error) {
	policies, err := c.enforcer.GetGroupingPolicy()
	if err != nil {
		return nil, fmt.Errorf("failed to get grouping policy: %w", err)
	}

	return policies, nil
}

// AddNamedPolicy adds a named policy rule of the given policy type.
func (c *Enforcer) AddNamedPolicy(_ context.Context, ptype string, params ...any) (bool, error) {
	added, err := c.enforcer.AddNamedPolicy(ptype, params...)
	if err != nil {
		return false, fmt.Errorf("failed to add named policy: %w", err)
	}

	return added, nil
}

// RemoveNamedPolicy removes a named policy rule of the given policy type.
func (c *Enforcer) RemoveNamedPolicy(_ context.Context, ptype string, params ...any) (bool, error) {
	removed, err := c.enforcer.RemoveNamedPolicy(ptype, params...)
	if err != nil {
		return false, fmt.Errorf("failed to remove named policy: %w", err)
	}

	return removed, nil
}

// GetNamedPolicy returns all policy rules of the given policy type.
func (c *Enforcer) GetNamedPolicy(ptype string) ([][]string, error) {
	policies, err := c.enforcer.GetNamedPolicy(ptype)
	if err != nil {
		return nil, fmt.Errorf("failed to get named policy: %w", err)
	}

	return policies, nil
}

// ClearPolicy removes all policies from the enforcer.
func (c *Enforcer) ClearPolicy(_ context.Context) {
	c.enforcer.ClearPolicy()
}
