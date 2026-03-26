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

	"github.com/minuk-dev/opampcommander/internal/domain/port"
)

var _ port.RBACEnforcerPort = (*CasbinEnforcer)(nil)

// CasbinEnforcer wraps a Casbin enforcer to implement RBACEnforcerPort.
type CasbinEnforcer struct {
	enforcer *casbin.Enforcer
}

// NewCasbinEnforcer creates a new CasbinEnforcer using a model config file.
// Policies are stored in-memory only.
func NewCasbinEnforcer(modelPath string) (*CasbinEnforcer, error) {
	e, err := casbin.NewEnforcer(modelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin enforcer: %w", err)
	}

	return &CasbinEnforcer{enforcer: e}, nil
}

// NewCasbinEnforcerWithAdapter creates a new CasbinEnforcer with a custom
// policy adapter (e.g., MongoDB) for persistent policy storage.
func NewCasbinEnforcerWithAdapter(modelPath string, adapter persist.Adapter) (*CasbinEnforcer, error) {
	m, err := model.NewModelFromFile(modelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load casbin model: %w", err)
	}

	e, err := casbin.NewEnforcer(m, adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin enforcer with adapter: %w", err)
	}

	return &CasbinEnforcer{enforcer: e}, nil
}

func (c *CasbinEnforcer) CheckPermission(_ context.Context, userID uuid.UUID, resource, action string) (bool, error) {
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

func (c *CasbinEnforcer) LoadPolicy(_ context.Context) error {
	if err := c.enforcer.LoadPolicy(); err != nil {
		return fmt.Errorf("failed to load policy: %w", err)
	}

	return nil
}

func (c *CasbinEnforcer) SavePolicy(_ context.Context) error {
	if err := c.enforcer.SavePolicy(); err != nil {
		return fmt.Errorf("failed to save policy: %w", err)
	}

	return nil
}

func (c *CasbinEnforcer) AddGroupingPolicy(_ context.Context, params ...interface{}) (bool, error) {
	added, err := c.enforcer.AddGroupingPolicy(params...)
	if err != nil {
		return false, fmt.Errorf("failed to add grouping policy: %w", err)
	}

	return added, nil
}

func (c *CasbinEnforcer) RemoveGroupingPolicy(_ context.Context, params ...interface{}) (bool, error) {
	removed, err := c.enforcer.RemoveGroupingPolicy(params...)
	if err != nil {
		return false, fmt.Errorf("failed to remove grouping policy: %w", err)
	}

	return removed, nil
}

func (c *CasbinEnforcer) GetGroupingPolicy() [][]string {
	policies, err := c.enforcer.GetGroupingPolicy()
	if err != nil {
		slog.Error("failed to get grouping policy", slog.Any("error", err))

		return nil
	}

	return policies
}

func (c *CasbinEnforcer) AddNamedPolicy(_ context.Context, ptype string, params ...interface{}) (bool, error) {
	added, err := c.enforcer.AddNamedPolicy(ptype, params...)
	if err != nil {
		return false, fmt.Errorf("failed to add named policy: %w", err)
	}

	return added, nil
}

func (c *CasbinEnforcer) RemoveNamedPolicy(_ context.Context, ptype string, params ...interface{}) (bool, error) {
	removed, err := c.enforcer.RemoveNamedPolicy(ptype, params...)
	if err != nil {
		return false, fmt.Errorf("failed to remove named policy: %w", err)
	}

	return removed, nil
}

func (c *CasbinEnforcer) GetNamedPolicy(ptype string) [][]string {
	policies, err := c.enforcer.GetNamedPolicy(ptype)
	if err != nil {
		slog.Error("failed to get named policy", slog.String("ptype", ptype), slog.Any("error", err))

		return nil
	}

	return policies
}
