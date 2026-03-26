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
	"github.com/google/uuid"

	userport "github.com/minuk-dev/opampcommander/internal/domain/user/port"
)

var _ userport.RBACEnforcerPort = (*Enforcer)(nil)

// Enforcer wraps a Casbin enforcer to implement RBACEnforcerPort.
type Enforcer struct {
	enforcer *casbin.Enforcer
}

// NewEnforcer creates a new Enforcer using a model config file.
// Policies are stored in-memory only.
func NewEnforcer(modelPath string) (*Enforcer, error) {
	e, err := casbin.NewEnforcer(modelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin enforcer: %w", err)
	}

	return &Enforcer{enforcer: e}, nil
}

// NewEnforcerFromModel creates a new Enforcer using a pre-loaded model.
// Policies are stored in-memory only.
func NewEnforcerFromModel(m model.Model) (*Enforcer, error) {
	e, err := casbin.NewEnforcer(m)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin enforcer: %w", err)
	}

	return &Enforcer{enforcer: e}, nil
}

// NewEnforcerWithAdapter creates a new Enforcer with a custom
// policy adapter (e.g., MongoDB) for persistent policy storage.
func NewEnforcerWithAdapter(modelPath string, adapter persist.Adapter) (*Enforcer, error) {
	m, err := model.NewModelFromFile(modelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load casbin model: %w", err)
	}

	e, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin enforcer with adapter: %w", err)
	}

	return &Enforcer{enforcer: e}, nil
}

// CheckPermission checks whether a user has the given permission on a resource.
func (c *Enforcer) CheckPermission(_ context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	allowed, err := c.enforcer.Enforce(userID.String(), resource, action)
	if err != nil {
		return false, fmt.Errorf("failed to check permission: %w", err)
	}

	slog.Debug("permission check",
		slog.String("user_id", userID.String()),
		slog.String("resource", resource),
		slog.String("action", action),
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
func (c *Enforcer) GetGroupingPolicy() [][]string {
	policies, err := c.enforcer.GetGroupingPolicy()
	if err != nil {
		slog.Error("failed to get grouping policy", slog.Any("error", err))

		return nil
	}

	return policies
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
func (c *Enforcer) GetNamedPolicy(ptype string) [][]string {
	policies, err := c.enforcer.GetNamedPolicy(ptype)
	if err != nil {
		slog.Error("failed to get named policy", slog.String("ptype", ptype), slog.Any("error", err))

		return nil
	}

	return policies
}
